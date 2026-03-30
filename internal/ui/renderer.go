package ui

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"multi-ping/internal/ping"
)

type Config struct {
	Checks   []ping.CheckKey
	Interval time.Duration
	Timeout  time.Duration
}

type Renderer struct {
	out      io.Writer
	cfg      Config
	interval time.Duration
}

func NewRenderer(out io.Writer, cfg Config) *Renderer {
	return &Renderer{
		out:      out,
		cfg:      cfg,
		interval: 250 * time.Millisecond,
	}
}

func (r *Renderer) Run(ctx context.Context, updates <-chan ping.Update) error {
	state := make(map[ping.CheckKey]ping.Update, len(r.cfg.Checks))
	for _, check := range r.cfg.Checks {
		state[check] = ping.Update{
			CheckKey: check,
			Status:   "STARTING",
			Note:     "waiting for first probe",
		}
	}

	if _, err := fmt.Fprint(r.out, "\x1b[?1049h\x1b[?25l"); err != nil {
		return err
	}
	defer fmt.Fprint(r.out, "\x1b[?25h\x1b[?1049l")

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	if err := r.draw(state); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			state[update.CheckKey] = update
		case <-ticker.C:
			if err := r.draw(state); err != nil {
				return err
			}
		}
	}
}

func (r *Renderer) draw(state map[ping.CheckKey]ping.Update) error {
	width, height := terminalSize()
	lines := r.renderLines(state, width, height)
	_, err := fmt.Fprintf(r.out, "\x1b[H\x1b[2J%s", strings.Join(lines, "\n"))
	return err
}

func (r *Renderer) renderLines(state map[ping.CheckKey]ping.Update, width, height int) []string {
	groups := groupByInterface(state)
	panes := make([][]string, 0, len(groups))
	for _, group := range groups {
		panes = append(panes, renderPane(group.interfaceName, group.updates, paneWidth(width, len(groups))))
	}

	upCount, downCount, startingCount := summarizeStates(state)
	header := []string{
		fit(fmt.Sprintf("multi-ping  checks:%d  up:%d  down:%d  starting:%d", len(r.cfg.Checks), upCount, downCount, startingCount), width),
		fit(fmt.Sprintf("interval:%s  timeout:%s  updated:%s  Ctrl+C exits", r.cfg.Interval, r.cfg.Timeout, time.Now().Format("2006-01-02 15:04:05")), width),
		strings.Repeat("=", max(1, width)),
	}

	body := composePaneGrid(panes, width)
	lines := append(header, body...)
	if len(lines) > height {
		if height < 2 {
			return lines[:height]
		}
		lines = lines[:height]
		lines[height-1] = fit("... output truncated; widen or heighten the terminal for more panes ...", width)
	}
	return lines
}

type paneGroup struct {
	interfaceName string
	updates       []ping.Update
}

func groupByInterface(state map[ping.CheckKey]ping.Update) []paneGroup {
	grouped := map[string][]ping.Update{}
	for _, update := range state {
		grouped[update.Interface] = append(grouped[update.Interface], update)
	}

	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)

	groups := make([]paneGroup, 0, len(names))
	for _, name := range names {
		updates := grouped[name]
		sort.Slice(updates, func(i, j int) bool {
			return updates[i].Target < updates[j].Target
		})
		groups = append(groups, paneGroup{
			interfaceName: name,
			updates:       updates,
		})
	}

	return groups
}

func composePaneGrid(panes [][]string, width int) []string {
	if len(panes) == 0 {
		return []string{"no panes"}
	}

	cols := columnsForWidth(width)
	gap := "  "
	var lines []string

	for start := 0; start < len(panes); start += cols {
		end := min(start+cols, len(panes))
		row := panes[start:end]
		maxHeight := 0
		for _, pane := range row {
			if len(pane) > maxHeight {
				maxHeight = len(pane)
			}
		}
		for lineIndex := 0; lineIndex < maxHeight; lineIndex++ {
			parts := make([]string, 0, len(row))
			for _, pane := range row {
				if lineIndex < len(pane) {
					parts = append(parts, pane[lineIndex])
					continue
				}
				parts = append(parts, strings.Repeat(" ", len(pane[0])))
			}
			lines = append(lines, strings.Join(parts, gap))
		}
		if end < len(panes) {
			lines = append(lines, "")
		}
	}

	return lines
}

func columnsForWidth(width int) int {
	switch {
	case width >= 180:
		return 3
	case width >= 118:
		return 2
	default:
		return 1
	}
}

func paneWidth(totalWidth, paneCount int) int {
	cols := columnsForWidth(totalWidth)
	if paneCount < cols {
		cols = paneCount
	}
	if cols < 1 {
		cols = 1
	}
	gapWidth := 2 * (cols - 1)
	width := (totalWidth - gapWidth) / cols
	if width < 28 {
		return max(1, width)
	}
	return width
}

func renderPane(interfaceName string, updates []ping.Update, width int) []string {
	width = max(width, 28)
	lines := []string{
		boxTop(interfaceName, width),
		boxLine("TARGET              STATE     RTT        LOSS    LAST CHECK", width),
		boxDivider(width),
	}

	for _, update := range updates {
		summary := fmt.Sprintf(
			"%-18s  %-8s  %-9s  %-6s  %-10s",
			fitField(update.Target, 18),
			update.Status,
			renderRTT(update),
			fmt.Sprintf("%5.1f%%", update.LossPercent),
			renderTimestamp(update.LastChecked),
		)
		lines = append(lines, boxLine(summary, width))
		lines = append(lines, boxLine("note: "+fitField(renderNote(update), width-10), width))
	}

	lines = append(lines, boxBottom(width))
	return lines
}

func summarizeStates(state map[ping.CheckKey]ping.Update) (up, down, starting int) {
	for _, update := range state {
		switch update.Status {
		case "UP":
			up++
		case "DOWN":
			down++
		default:
			starting++
		}
	}
	return up, down, starting
}

func renderRTT(update ping.Update) string {
	if !update.Reachable || update.RTT <= 0 {
		return "-"
	}
	return update.RTT.Round(time.Microsecond).String()
}

func renderTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format("15:04:05")
}

func renderNote(update ping.Update) string {
	if update.Note != "" {
		return update.Note
	}
	if update.Reachable {
		return "reply received"
	}
	return "probe failed"
}

func boxTop(title string, width int) string {
	inner := width - 2
	label := " " + fitField(title, inner-2) + " "
	left := max(0, (inner-len(label))/2)
	right := max(0, inner-len(label)-left)
	return "+" + strings.Repeat("-", left) + label + strings.Repeat("-", right) + "+"
}

func boxDivider(width int) string {
	return "+" + strings.Repeat("-", width-2) + "+"
}

func boxBottom(width int) string {
	return "+" + strings.Repeat("-", width-2) + "+"
}

func boxLine(content string, width int) string {
	inner := width - 2
	return "|" + fit(content, inner) + "|"
}

func fit(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > width {
		if width == 1 {
			return string(runes[:1])
		}
		return string(runes[:width-1]) + "~"
	}
	return string(runes) + strings.Repeat(" ", width-len(runes))
}

func fitField(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "~"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
