// Package styles define styling and theming for the project.
package styles

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/ChxisB/talon/deps/render/v2/ansi"
	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/deps/ui/core/v2/filepicker"
	"github.com/ChxisB/talon/deps/ui/core/v2/help"
	"github.com/ChxisB/talon/deps/ui/core/v2/textarea"
	"github.com/ChxisB/talon/deps/ui/core/v2/textinput"
	"github.com/ChxisB/talon/internal/ui/diffview"
	"github.com/alecthomas/chroma/v2"
)

const (
	CheckIcon       string = "✓"
	SpinnerIcon     string = "⋯"
	LoadingIcon     string = "⟳"
	ModelIcon       string = "◇"
	HypercreditIcon string = "◆"

	ArrowRightIcon string = "→"

	ToolPending string = "●"
	ToolSuccess string = "✓"
	ToolError   string = "×"

	RadioOn  string = "◉"
	RadioOff string = "○"

	BorderThin  string = "│"
	BorderThick string = "▌"

	SectionSeparator string = "─"

	TodoCompletedIcon  string = "✓"
	TodoPendingIcon    string = "•"
	TodoInProgressIcon string = "→"

	ImageIcon string = "■"
	TextIcon  string = "≡"
	SkillIcon string = "▲"

	ScrollbarThumb string = "┃"
	ScrollbarTrack string = "│"

	LSPErrorIcon   string = "E"
	LSPWarningIcon string = "W"
	LSPInfoIcon    string = "I"
	LSPHintIcon    string = "H"
)

const (
	defaultMargin     = 2
	defaultListIndent = 2
)

type Styles struct {
	// Header
	Header struct {
		Talon           style.Style // Style for "Talon" label
		Diagonals         style.Style // Style for diagonal separators (╱)
		Percentage        style.Style // Style for context percentage
		Hypercredit       style.Style // Style for Hypercredit count (◆ N)
		Keystroke         style.Style // Style for keystroke hints (e.g., "ctrl+d")
		KeystrokeTip      style.Style // Style for keystroke action text (e.g., "open", "close")
		WorkingDir        style.Style // Style for current working directory
		Separator         style.Style // Style for separator dots (•)
		Wrapper           style.Style // Outer container for the entire header row
		LogoGradCanvas    style.Style // Canvas for the compact "TALON" gradient
		LogoGradFromColor color.Color    // "TALON" wordmark gradient start
		LogoGradToColor   color.Color    // "TALON" wordmark gradient end
	}

	CompactDetails struct {
		View    style.Style
		Version style.Style
		Title   style.Style
	}

	// Tool calls
	ToolCallSuccess style.Style

	// Text selection
	TextSelection style.Style

	// Markdown & Chroma
	Markdown      ansi.StyleConfig
	QuietMarkdown ansi.StyleConfig

	// Inputs
	TextInput textinput.Styles

	// Help
	Help help.Styles

	// Diff
	Diff diffview.Style

	// FilePicker
	FilePicker filepicker.Styles

	// Buttons
	Button struct {
		Focused style.Style
		Blurred style.Style
	}

	// Editor
	Editor struct {
		Textarea textarea.Styles

		// Normal mode prompt (default "::: ").
		PromptNormalFocused style.Style
		PromptNormalBlurred style.Style

		// YOLO mode prompt (" ! " icon + ":::" dots).
		PromptYoloIconFocused style.Style
		PromptYoloIconBlurred style.Style
		PromptYoloDotsFocused style.Style
		PromptYoloDotsBlurred style.Style
	}

	// Radio
	Radio struct {
		On    style.Style
		Off   style.Style
		Label style.Style // Text next to a radio button
	}

	// Background
	Background color.Color

	// Logo
	Logo struct {
		FieldColor         color.Color
		TitleColorA        color.Color
		TitleColorB        color.Color
		BrandColor         color.Color
		VersionColor       color.Color
		SmallBrand         style.Style // "Talon" label in SmallRender
		SmallDiagonals     style.Style // Diagonal line fill in SmallRender
		GradCanvas         style.Style // Blank canvas for gradient painting
		SmallGradFromColor color.Color    // Small "Talon" wordmark gradient start
		SmallGradToColor   color.Color    // Small "Talon" wordmark gradient end
	}

	// Working indicator gradient (spinners/shimmers on assistant "thinking",
	// tool-call pending, CLI generating, startup).
	WorkingGradFromColor color.Color
	WorkingGradToColor   color.Color
	WorkingLabelColor    color.Color // Label text color next to the indicator

	// Section Title
	Section struct {
		Title style.Style
		Line  style.Style
	}

	// Initialize
	Initialize struct {
		Header  style.Style
		Content style.Style
		Accent  style.Style
	}

	// LSP
	LSP struct {
		ErrorDiagnostic   style.Style
		WarningDiagnostic style.Style
		HintDiagnostic    style.Style
		InfoDiagnostic    style.Style
	}

	// Sidebar
	Sidebar struct {
		SessionTitle style.Style // Current session title at top of sidebar
		WorkingDir   style.Style // Working directory path (PrettyPath)
	}

	// ModelInfo (model name, provider, reasoning, token/cost summary)
	ModelInfo struct {
		Icon                 style.Style // Model icon (◇)
		Name                 style.Style // Model name text
		Provider             style.Style // "via <provider>" text
		ProviderFallback     style.Style // Provider on its own second line
		Reasoning            style.Style // Reasoning effort text
		TokenCount           style.Style // "(42K)" token count
		TokenPercentage      style.Style // "42%" percent of context window
		EstimatedUsagePrefix style.Style // "~" prefix for estimated usage
		Cost                 style.Style // "$0.42" cost readout
		HypercreditIcon      style.Style // Hypercredit icon (◆)
		HypercreditText      style.Style // Remaining Hypercredits text
		SaverEnabled         style.Style // "on" / token saver level
		SaverDisabled        style.Style // "off" token saver status
	}

	// Resource styles the LSP/MCP/skills sidebar lists: their heading,
	// each row's status icon, name, status text, and truncation hints.
	Resource struct {
		Heading         style.Style // Section header ("LSPs", "MCPs", "Skills")
		Name            style.Style // Resource name (e.g. "gopls")
		StatusText      style.Style // Row status description (e.g. "starting...")
		OfflineIcon     style.Style // Offline/unstarted/stopped status icon
		DisabledIcon    style.Style // Disabled status icon
		BusyIcon        style.Style // Busy/starting status icon
		ErrorIcon       style.Style // Error status icon
		OnlineIcon      style.Style // Online/ready status icon
		AdditionalText  style.Style // "None" and "…and N more" text
		CapabilityCount style.Style // "N tools" / "N prompts" / "N resources"
		RowTitleBase    style.Style // Base style applied over row titles in common.Status
		RowDescBase     style.Style // Base style applied over row descriptions in common.Status
		DefaultTitleFg  color.Color    // Default title color when opt is zero
		DefaultDescFg   color.Color    // Default description color when opt is zero
	}

	// Files
	Files struct {
		Path           style.Style
		Additions      style.Style
		Deletions      style.Style
		SectionTitle   style.Style // "Modified Files" heading
		EmptyMessage   style.Style // "None" placeholder when no files
		TruncationHint style.Style // "…and N more" message
	}

	// Chat
	// Messages - chat message item styles
	Messages struct {
		UserBlurred      style.Style
		UserFocused      style.Style
		AssistantBlurred style.Style
		AssistantFocused style.Style
		NoContent        style.Style
		Thinking         style.Style
		ErrorTag         style.Style
		ErrorTitle       style.Style
		ErrorDetails     style.Style
		ToolCallFocused  style.Style
		ToolCallCompact  style.Style
		ToolCallBlurred  style.Style
		SectionHeader    style.Style

		// Thinking section styles
		ThinkingBox            style.Style // Background for thinking content
		ThinkingTruncationHint style.Style // "… (N lines hidden)" hint
		ThinkingFooterTitle    style.Style // "Thought for" text
		ThinkingFooterDuration style.Style // Duration value
		AssistantInfoIcon      style.Style
		AssistantInfoModel     style.Style
		AssistantInfoProvider  style.Style
		AssistantInfoDuration  style.Style
		AssistantCanceled      style.Style // Italic "Canceled" footer
	}

	// Tool - styles for tool call rendering
	Tool struct {
		// Icon styles with tool status
		IconPending   style.Style
		IconSuccess   style.Style
		IconError     style.Style
		IconCancelled style.Style

		// Tool name styles
		NameNormal style.Style // Top-level tool name
		NameNested style.Style // Nested child tool name (inside Agent/Agentic Fetch)

		// Parameter list styles
		ParamMain style.Style
		ParamKey  style.Style

		// Content rendering styles
		ContentLine           style.Style // Individual content line with background and width
		ContentTruncation     style.Style // Truncation message "… (N lines)"
		ContentCodeLine       style.Style // Code line with background and width
		ContentCodeTruncation style.Style // Code truncation message with bgBase
		ContentCodeBg         color.Color    // Background color for syntax highlighting
		Body                  style.Style // Body content padding (PaddingLeft(2))

		// Deprecated - kept for backward compatibility
		ContentBg         style.Style // Content background
		ContentText       style.Style // Content text
		ContentLineNumber style.Style // Line numbers in code

		// State message styles
		StateWaiting   style.Style // "Waiting for tool response..."
		StateCancelled style.Style // "Canceled."

		// Error styles
		ErrorTag     style.Style // ERROR tag
		ErrorMessage style.Style // Error message text

		// Warning styles (used for permission denied)
		WarnTag     style.Style // WARN tag
		WarnMessage style.Style // Warning message text

		// Diff styles
		DiffTruncation style.Style // Diff truncation message with padding

		// Multi-edit note styles
		NoteTag     style.Style // NOTE tag (yellow background)
		NoteMessage style.Style // Note message text

		// Job header styles (for bash jobs)
		JobIconPending style.Style // Pending job icon (green dark)
		JobIconError   style.Style // Error job icon (red dark)
		JobIconSuccess style.Style // Success job icon (green)
		JobToolName    style.Style // Job tool name "Bash" (blue)
		JobAction      style.Style // Action text (Start, Output, Kill)
		JobPID         style.Style // PID text
		JobDescription style.Style // Description text

		// Agent task styles
		AgentTaskTag style.Style // Agent task tag (blue background, bold)
		AgentPrompt  style.Style // Agent prompt text

		// Agentic fetch styles
		AgenticFetchPromptTag style.Style // Agentic fetch prompt tag (green background, bold)

		// Todo styles
		TodoRatio          style.Style // Todo ratio (e.g., "2/5")
		TodoCompletedIcon  style.Style // Completed todo icon
		TodoInProgressIcon style.Style // In-progress todo icon
		TodoPendingIcon    style.Style // Pending todo icon
		TodoStatusNote     style.Style // " · completed N" / " · starting task" trailing note
		TodoItem           style.Style // Default body text for todo list items
		TodoJustStarted    style.Style // Text of the just-started todo in tool-call bodies

		// MCP tools
		MCPName     style.Style // The mcp name
		MCPToolName style.Style // The mcp tool name
		MCPArrow    style.Style // The mcp arrow icon

		// Images and external resources
		ResourceLoadedText      style.Style
		ResourceLoadedIndicator style.Style
		ResourceName            style.Style
		ResourceSize            style.Style
		MediaType               style.Style

		// Hooks
		HookLabel        style.Style // "Hook" label
		HookName         style.Style // Hook command name
		HookMatcher      style.Style // Matcher regex pattern
		HookArrow        style.Style // Arrow indicator
		HookDetail       style.Style // Decision detail text
		HookOK           style.Style // "OK" status
		HookDenied       style.Style // "Denied" status
		HookDeniedLabel  style.Style // "Hook" label when denied
		HookDeniedReason style.Style // Denied reason text
		HookRewrote      style.Style // "Rewrote Input" indicator

		// Action verb colors for tool-call headers.
		ActionCreate  style.Style // Constructive actions (e.g. "Add", "Create")
		ActionDestroy style.Style // Destructive actions (e.g. "Remove", "Delete")

		// Tool result helpers.
		ResultEmpty      style.Style // "No results" placeholder
		ResultTruncation style.Style // "… and N more" truncation line
		ResultItemName   style.Style // Item name (left column in result lists)
		ResultItemDesc   style.Style // Item description (right column)
	}

	// Dialog styles
	Dialog struct {
		Title              style.Style
		TitleText          style.Style
		TitleError         style.Style
		TitleAccent        style.Style
		TitleLineBase      style.Style // Base for the gradient ╱╱╱ next to dialog titles
		TitleGradFromColor color.Color    // Default dialog title ╱╱╱ gradient start
		TitleGradToColor   color.Color    // Default dialog title ╱╱╱ gradient end
		// View is the main content area style.
		View          style.Style
		PrimaryText   style.Style
		SecondaryText style.Style
		// HelpView is the line that contains the help.
		HelpView style.Style
		Help     struct {
			Ellipsis       style.Style
			ShortKey       style.Style
			ShortDesc      style.Style
			ShortSeparator style.Style
			FullKey        style.Style
			FullDesc       style.Style
			FullSeparator  style.Style
		}

		NormalItem   style.Style
		SelectedItem style.Style
		InputPrompt  style.Style

		List style.Style

		Spinner style.Style

		// ContentPanel is used for content blocks with subtle background.
		ContentPanel style.Style

		// Scrollbar styles for scrollable content.
		ScrollbarThumb style.Style
		ScrollbarTrack style.Style

		// Arguments
		Arguments struct {
			Content                  style.Style
			Description              style.Style
			InputLabelBlurred        style.Style
			InputLabelFocused        style.Style
			InputRequiredMarkBlurred style.Style
			InputRequiredMarkFocused style.Style
		}

		// ListItem styles the info-text rendered alongside list items (commands,
		// models, reasoning options). Sessions have their own overrides below.
		ListItem struct {
			InfoBlurred style.Style
			InfoFocused style.Style
		}

		Models struct {
			ConfiguredText style.Style // "Configured" badge shown on the ModelGroup header
		}

		Permissions struct {
			KeyText   style.Style // Left key cell of a key/value row
			ValueText style.Style // Right value cell of a key/value row
			ParamsBg  color.Color    // Background color behind highlighted JSON parameters
		}

		Quit struct {
			Content style.Style // Wrapper for the quit dialog's inner content
			Frame   style.Style // Outer rounded border framing the quit dialog
		}

		APIKey struct {
			Spinner style.Style // Loading spinner while validating the key
		}

		OAuth struct {
			Spinner      style.Style // Loading spinner
			Instructions style.Style // Emphasized instruction text
			UserCode     style.Style // Prominent user code display
			Success      style.Style // Positive status text (e.g. "Authentication successful!")
			Link         style.Style // Underlined verification URL
			Enter        style.Style // "enter" keyword highlight in instructions
			ErrorText    style.Style // Error message when authentication fails
			StatusText   style.Style // Narrative status text ("Initializing...", "Verifying...", etc.)
			UserCodeBg   color.Color    // Background color of the centered user-code box
		}

		ImagePreview style.Style

		Sessions struct {
			// styles for when we are in delete mode
			DeletingView                   style.Style
			DeletingItemFocused            style.Style
			DeletingItemBlurred            style.Style
			DeletingTitle                  style.Style
			DeletingMessage                style.Style
			DeletingTitleGradientFromColor color.Color
			DeletingTitleGradientToColor   color.Color

			// styles for when we are in update mode
			RenamingView                   style.Style
			RenamingingItemFocused         style.Style
			RenamingItemBlurred            style.Style
			RenamingingTitle               style.Style
			RenamingingMessage             style.Style
			RenamingTitleGradientFromColor color.Color
			RenamingTitleGradientToColor   color.Color
			RenamingPlaceholder            style.Style

			InfoBlurred style.Style // Timestamp text on unfocused session items
			InfoFocused style.Style // Timestamp text on the focused session item
		}
	}

	// Status bar and help
	Status struct {
		Help style.Style

		ErrorIndicator   style.Style
		WarnIndicator    style.Style
		InfoIndicator    style.Style
		UpdateIndicator  style.Style
		SuccessIndicator style.Style

		ErrorMessage   style.Style
		WarnMessage    style.Style
		InfoMessage    style.Style
		UpdateMessage  style.Style
		SuccessMessage style.Style
	}

	// Completions popup styles
	Completions struct {
		Normal  style.Style
		Focused style.Style
		Match   style.Style
	}

	// Attachments styles
	Attachments struct {
		Normal   style.Style
		Image    style.Style
		Text     style.Style
		Skill    style.Style
		Deleting style.Style
	}

	// Pills styles for todo/queue pills
	Pills struct {
		Base               style.Style // Base pill style with padding
		Focused            style.Style // Focused pill with visible border
		Blurred            style.Style // Blurred pill with hidden border
		QueueItemPrefix    style.Style // Prefix for queue list items
		QueueItemText      style.Style // Queue list item body text
		QueueLabel         style.Style // "N Queued" label text
		QueueIconBase      style.Style // Base style for queue gradient triangles
		QueueGradFromColor color.Color    // Start color for queue indicator gradient
		QueueGradToColor   color.Color    // End color for queue indicator gradient
		TodoLabel          style.Style // "To-Do" label
		TodoProgress       style.Style // Todo ratio (e.g. "2/5")
		TodoCurrentTask    style.Style // Current in-progress task name
		TodoSpinner        style.Style // Todo spinner style
		HelpKey            style.Style // Keystroke hint style
		HelpText           style.Style // Help action text style
		Area               style.Style // Pills area container
	}
}

// ChromaTheme converts the current markdown chroma styles to a chroma
// StyleEntries map.
func (s *Styles) ChromaTheme() chroma.StyleEntries {
	rules := s.Markdown.CodeBlock

	return chroma.StyleEntries{
		chroma.Text:                chromaStyle(rules.Chroma.Text),
		chroma.Error:               chromaStyle(rules.Chroma.Error),
		chroma.Comment:             chromaStyle(rules.Chroma.Comment),
		chroma.CommentPreproc:      chromaStyle(rules.Chroma.CommentPreproc),
		chroma.Keyword:             chromaStyle(rules.Chroma.Keyword),
		chroma.KeywordReserved:     chromaStyle(rules.Chroma.KeywordReserved),
		chroma.KeywordNamespace:    chromaStyle(rules.Chroma.KeywordNamespace),
		chroma.KeywordType:         chromaStyle(rules.Chroma.KeywordType),
		chroma.Operator:            chromaStyle(rules.Chroma.Operator),
		chroma.Punctuation:         chromaStyle(rules.Chroma.Punctuation),
		chroma.Name:                chromaStyle(rules.Chroma.Name),
		chroma.NameBuiltin:         chromaStyle(rules.Chroma.NameBuiltin),
		chroma.NameTag:             chromaStyle(rules.Chroma.NameTag),
		chroma.NameAttribute:       chromaStyle(rules.Chroma.NameAttribute),
		chroma.NameClass:           chromaStyle(rules.Chroma.NameClass),
		chroma.NameConstant:        chromaStyle(rules.Chroma.NameConstant),
		chroma.NameDecorator:       chromaStyle(rules.Chroma.NameDecorator),
		chroma.NameException:       chromaStyle(rules.Chroma.NameException),
		chroma.NameFunction:        chromaStyle(rules.Chroma.NameFunction),
		chroma.NameOther:           chromaStyle(rules.Chroma.NameOther),
		chroma.Literal:             chromaStyle(rules.Chroma.Literal),
		chroma.LiteralNumber:       chromaStyle(rules.Chroma.LiteralNumber),
		chroma.LiteralDate:         chromaStyle(rules.Chroma.LiteralDate),
		chroma.LiteralString:       chromaStyle(rules.Chroma.LiteralString),
		chroma.LiteralStringEscape: chromaStyle(rules.Chroma.LiteralStringEscape),
		chroma.GenericDeleted:      chromaStyle(rules.Chroma.GenericDeleted),
		chroma.GenericEmph:         chromaStyle(rules.Chroma.GenericEmph),
		chroma.GenericInserted:     chromaStyle(rules.Chroma.GenericInserted),
		chroma.GenericStrong:       chromaStyle(rules.Chroma.GenericStrong),
		chroma.GenericSubheading:   chromaStyle(rules.Chroma.GenericSubheading),
		chroma.Background:          chromaStyle(rules.Chroma.Background),
	}
}

// DialogHelpStyles returns the styles for dialog help.
func (s *Styles) DialogHelpStyles() help.Styles {
	return help.Styles(s.Dialog.Help)
}

// hex returns a pointer to the "#rrggbb" representation of c. It's used to
// satisfy glamour's string-pointer API when configuring markdown colors
// from the theme palette.
func hex(c color.Color) *string {
	r, g, b, _ := c.RGBA()
	s := fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	return &s
}

func chromaStyle(style ansi.StylePrimitive) string {
	var s strings.Builder

	if style.Color != nil {
		s.WriteString(*style.Color)
	}
	if style.BackgroundColor != nil {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("bg:")
		s.WriteString(*style.BackgroundColor)
	}
	if style.Italic != nil && *style.Italic {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("italic")
	}
	if style.Bold != nil && *style.Bold {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("bold")
	}
	if style.Underline != nil && *style.Underline {
		if s.Len() > 0 {
			s.WriteString(" ")
		}
		s.WriteString("underline")
	}

	return s.String()
}
