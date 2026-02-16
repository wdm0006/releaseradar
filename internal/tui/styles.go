package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette — deep space theme with warm accents
	colorPrimary    = lipgloss.Color("#8B5CF6") // vibrant purple
	colorAccent     = lipgloss.Color("#F59E0B") // amber/gold for highlights
	colorCyan       = lipgloss.Color("#06B6D4") // cyan for secondary info
	colorMuted      = lipgloss.Color("#6B7280") // gray text
	colorSubtle     = lipgloss.Color("#9CA3AF") // lighter gray
	colorSuccess    = lipgloss.Color("#34D399") // green
	colorError      = lipgloss.Color("#F87171") // red
	colorFg         = lipgloss.Color("#F3F4F6") // near-white
	colorDimFg      = lipgloss.Color("#D1D5DB") // dim white
	colorBorder     = lipgloss.Color("#4B5563") // medium gray borders
	colorDimBorder  = lipgloss.Color("#374151") // subtle borders
	colorPanelBg    = lipgloss.Color("#111827") // dark panel background
	colorHeaderBg   = lipgloss.Color("#1E1B4B") // deep indigo for headers
	colorSelectedBg = lipgloss.Color("#4C1D95") // selected row background
	colorTableRowBg = lipgloss.Color("#1F2937") // alternate row background

	// Tab bar
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Padding(0, 2)

	tabBarStyle = lipgloss.NewStyle().
			Background(colorPanelBg).
			PaddingBottom(0).
			MarginBottom(0)

	tabGapStyle = lipgloss.NewStyle().
			Foreground(colorDimBorder)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 1)

	statusAccentStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	// Panels
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDimBorder).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Title
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// Section header inside detail panels
	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	// Metadata labels and values in detail panels
	metaLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	metaValueStyle = lipgloss.NewStyle().
			Foreground(colorDimFg)

	// Badge styles for repo stats
	badgeStyle = lipgloss.NewStyle().
			Foreground(colorPanelBg).
			Background(colorPrimary).
			Bold(true).
			Padding(0, 1)

	badgeCyanStyle = lipgloss.NewStyle().
			Foreground(colorPanelBg).
			Background(colorCyan).
			Bold(true).
			Padding(0, 1)

	badgeAmberStyle = lipgloss.NewStyle().
			Foreground(colorPanelBg).
			Background(colorAccent).
			Bold(true).
			Padding(0, 1)

	// Help
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Modal overlay
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 3).
			Width(54)

	// Muted text
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)

	// Loading screen
	logoStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	logoSubtitleStyle = lipgloss.NewStyle().
				Foreground(colorAccent)

	progressBarFilledStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	progressBarEmptyStyle = lipgloss.NewStyle().
				Foreground(colorDimBorder)

	progressTextStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true)

	progressRepoStyle = lipgloss.NewStyle().
				Foreground(colorCyan)

	progressCountStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Bold(true)

	// Release detail
	releaseNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	releaseTagStyle = lipgloss.NewStyle().
			Foreground(colorCyan)

	releaseDateStyle = lipgloss.NewStyle().
				Foreground(colorSubtle)

	releaseAuthorStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	releaseDividerStyle = lipgloss.NewStyle().
				Foreground(colorDimBorder)

	// New release indicator
	newIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)
)
