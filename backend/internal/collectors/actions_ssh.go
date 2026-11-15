package collectors

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// sshActor runs power, process, service and fan actions on Linux and UNAS
// hosts. Each command comes from a fixed template; only values that
// ValidateAction accepted are interpolated.
type sshActor struct {
	conn     *SSHConnection
	username string
}

func (s *sshActor) Run(ctx context.Context, a Action) (string, error) {
	cmd, err := sshCommand(a, s.username)
	if err != nil {
		return "", err
	}
	defer s.conn.Close()
	if a.Name == "kill" {
		// The PID may belong to another process since the last poll.
		comm, err := s.conn.RunAction(ctx, procCommCmd(a.PID))
		if err != nil {
			return "", err
		}
		if err := checkProcComm(a, comm); err != nil {
			return "", err
		}
	}
	out, err := s.conn.RunAction(ctx, cmd)
	return strings.TrimSpace(out), err
}

// procCommCmd prints the comm of pid, or nothing if the process is gone.
func procCommCmd(pid int) string {
	return fmt.Sprintf(`sh -c 'cat /proc/%d/comm 2>/dev/null; true'`, pid)
}

// checkProcComm compares /proc/<pid>/comm with the name from the last poll.
// The collector reads the name from the comm field of /proc/<pid>/stat,
// which is the same 15-character comm, so they compare exactly.
func checkProcComm(a Action, comm string) error {
	if strings.TrimSuffix(comm, "\n") != a.ProcName {
		return conflict("process %d changed since the last poll; refresh and retry", a.PID)
	}
	return nil
}

// sshCommand builds the full remote command for a validated action. The
// script runs under "sh -c" because the login shell may be zsh. Each
// privileged step uses "sudo -n <command>" (no sudo for root), so a narrow
// sudoers rule can allow exactly these commands.
func sshCommand(a Action, username string) (string, error) {
	var script string
	switch a.Name {
	case "reboot", "poweroff":
		script = powerScript(a.Name)
	case "kill":
		script = fmt.Sprintf(`$S kill -s %s %d`, a.Params.Signal, a.PID)
	case "service_start", "service_stop", "service_restart":
		script = fmt.Sprintf(`$S systemctl %s %s`, strings.TrimPrefix(a.Name, "service_"), a.Target)
	case "fan_set":
		script = fanSetScript(*a.Params.Percent)
	case "fan_auto":
		script = fanAutoScript
	default:
		return "", fmt.Errorf("action %q has no SSH command", a.Name)
	}
	sudo := "sudo -n"
	if username == "root" {
		sudo = ""
	}
	return fmt.Sprintf(`sh -c 'S="%s"; %s'`, sudo, script), nil
}

// powerScript checks that sudo permits the command, then runs it detached
// after a short delay so the SSH session can return before the host goes
// down.
func powerScript(verb string) string {
	return fmt.Sprintf(`if [ -n "$S" ]; then $S -l systemctl %[1]s >/dev/null || { echo "sudo -n does not permit: systemctl %[1]s" >&2; exit 1; }; fi; `+
		`nohup sh -c "sleep 2; $S systemctl %[1]s" >/dev/null 2>&1 </dev/null & echo "%[1]s scheduled in 2s"`, verb)
}

// pwmValue converts a duty-cycle percent to the hwmon pwm range 0..255.
func pwmValue(percent float64) int {
	return int(math.Round(percent * 255 / 100))
}

// fanSetScript puts the pwmfan device under manual control and writes the
// duty cycle. It records what it changes in the state file, so fan_auto can
// restore it. A second call changes nothing more and keeps the record.
func fanSetScript(percent float64) string {
	return fanScriptHeader + `
[ -e "$STATE" ] || $S tee "$STATE" </dev/null >/dev/null || exit 1
if systemctl is-active -q nvfancontrol 2>/dev/null; then
  echo nvfancontrol | $S tee -a "$STATE" >/dev/null || exit 1
  $S systemctl stop nvfancontrol || exit 1
fi
CD=
for c in /sys/class/thermal/cooling_device*; do
  [ "$(cat "$c/type" 2>/dev/null)" = pwm-fan ] && { CD=${c##*/}; break; }
done
if [ -n "$CD" ]; then
  for z in /sys/class/thermal/thermal_zone*; do
    hit=
    for l in "$z"/cdev*; do
      [ -L "$l" ] && [ "$(basename "$(readlink "$l")")" = "$CD" ] && hit=1
    done
    [ -n "$hit" ] || continue
    old=$(cat "$z/policy")
    [ "$old" = user_space ] && continue
    echo "zone $z $old" | $S tee -a "$STATE" >/dev/null || exit 1
    echo user_space | $S tee "$z/policy" >/dev/null || exit 1
  done
fi
if [ -e "$HW/pwm1_enable" ]; then
  old=$(cat "$HW/pwm1_enable")
  if [ "$old" != 1 ]; then
    echo "enable $HW/pwm1_enable $old" | $S tee -a "$STATE" >/dev/null || exit 1
    echo 1 | $S tee "$HW/pwm1_enable" >/dev/null || exit 1
  fi
fi
echo ` + fmt.Sprint(pwmValue(percent)) + ` | $S tee "$HW/pwm1" >/dev/null || exit 1
echo "pwm1=` + fmt.Sprint(pwmValue(percent)) + `"`
}

// fanAutoScript restores what fanSetScript recorded and removes the state
// file. Zones go back to their recorded policy (step_wise if none) and
// pwm1_enable goes back to its recorded value.
const fanAutoScript = `STATE=` + fanStateFile + `
[ -e "$STATE" ] || { echo "fan already in auto mode"; exit 0; }
while read -r kind path policy; do
  case "$kind" in
    zone) echo "${policy:-step_wise}" | $S tee "$path/policy" >/dev/null || exit 1 ;;
    enable) echo "$policy" | $S tee "$path" >/dev/null || exit 1 ;;
    nvfancontrol) $S systemctl start nvfancontrol || exit 1 ;;
  esac
done < "$STATE"
$S rm -f "$STATE" || exit 1
echo "fan control restored to auto"`

const fanScriptHeader = `STATE=` + fanStateFile + `
HW=` + fanHwmonDirCmd + `
[ -n "$HW" ] || { echo "no ` + fanHwmonName + ` hwmon device" >&2; exit 1; }`
