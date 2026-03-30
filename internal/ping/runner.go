package ping

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CheckKey struct {
	Interface string
	Target    string
}

type Update struct {
	CheckKey
	Sequence    int
	Sent        int
	Received    int
	LossPercent float64
	Status      string
	Reachable   bool
	RTT         time.Duration
	LastChecked time.Time
	LastSuccess time.Time
	Note        string
}

type Runner struct {
	CheckKey
	Interval time.Duration
	Timeout  time.Duration
	Out      chan<- Update
}

var rttPattern = regexp.MustCompile(`time[=<]([0-9.]+)\s*ms`)

func (r Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	update := Update{
		CheckKey: r.CheckKey,
		Status:   "STARTING",
		Note:     "waiting for first probe",
	}
	r.publish(update)

	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	for {
		update = r.probe(ctx, update)
		r.publish(update)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r Runner) probe(ctx context.Context, previous Update) Update {
	update := previous
	update.Sequence++
	update.Sent++
	update.LastChecked = time.Now()

	cmdCtx, cancel := context.WithTimeout(ctx, r.Timeout+time.Second)
	defer cancel()

	args := []string{
		"-n",
		"-c", "1",
		"-W", strconv.Itoa(timeoutSeconds(r.Timeout)),
		"-I", r.Interface,
		r.Target,
	}

	output, err := exec.CommandContext(cmdCtx, "ping", args...).CombinedOutput()
	body := string(output)
	if cmdCtx.Err() == context.DeadlineExceeded {
		update.Status = "DOWN"
		update.Reachable = false
		update.RTT = 0
		update.LossPercent = computeLoss(update.Sent, update.Received)
		update.Note = fmt.Sprintf("probe timed out after %s", r.Timeout.Round(time.Millisecond))
		return update
	}

	if err != nil {
		update.Status = "DOWN"
		update.Reachable = false
		update.RTT = 0
		update.LossPercent = computeLoss(update.Sent, update.Received)
		update.Note = summarizeFailure(body, err)
		return update
	}

	update.Received++
	update.Reachable = true
	update.Status = "UP"
	update.LastSuccess = update.LastChecked
	update.RTT = parseRTT(body)
	update.LossPercent = computeLoss(update.Sent, update.Received)
	if update.RTT > 0 {
		update.Note = fmt.Sprintf("reply in %s", update.RTT.Round(time.Microsecond))
	} else {
		update.Note = "reply received"
	}
	return update
}

func (r Runner) publish(update Update) {
	select {
	case r.Out <- update:
	default:
	}
}

func timeoutSeconds(d time.Duration) int {
	seconds := int(d / time.Second)
	if d%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func computeLoss(sent, received int) float64 {
	if sent == 0 {
		return 0
	}
	return float64(sent-received) / float64(sent) * 100
}

func parseRTT(output string) time.Duration {
	matches := rttPattern.FindStringSubmatch(output)
	if len(matches) != 2 {
		return 0
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}
	return time.Duration(value * float64(time.Millisecond))
}

func summarizeFailure(output string, fallback error) string {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		line = strings.TrimSpace(stripStatistics(line))
		if line == "" {
			continue
		}
		return line
	}
	return fallback.Error()
}

func stripStatistics(line string) string {
	if strings.Contains(line, "packets transmitted") || strings.Contains(line, "packet loss") {
		return ""
	}
	if bytes.HasPrefix([]byte(line), []byte("rtt ")) {
		return ""
	}
	return line
}
