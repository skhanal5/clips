package edit

// BackgroundType defines the background style for the output video.
type BackgroundType int

const (
	// BlackScreen uses a solid black background.
	BlackScreen BackgroundType = iota
	// BlurredVideo uses the input video blurred as background.
	BlurredVideo
	// StaticImage uses a static image as background.
	StaticImage
)

// Size represents a video dimension.
type Size struct {
	Width  int
	Height int
}

// Options controls the video editing behavior.
type Options struct {
	Background     BackgroundType
	BgImagePath    string
	ForegroundSize Size
	Title          string
}

// Option configures an Options.
type Option func(*Options)

// WithBackground sets the background type.
func WithBackground(bg BackgroundType) Option {
	return func(o *Options) {
		o.Background = bg
	}
}

// WithBgImage sets the background image path (used with StaticImage).
func WithBgImage(path string) Option {
	return func(o *Options) {
		o.BgImagePath = path
	}
}

// WithForegroundSize sets the clipped foreground video size.
func WithForegroundSize(width, height int) Option {
	return func(o *Options) {
		o.ForegroundSize = Size{Width: width, Height: height}
	}
}

// TemplateType is a shorthand for common edit configurations.
type TemplateType string

const (
	// TemplateBlurred uses a blurred video background.
	TemplateBlurred TemplateType = "blurred"
	// TemplateBlack uses a solid black background.
	TemplateBlack TemplateType = "black"
	// TemplateImage uses a static image background.
	TemplateImage TemplateType = "image"
)

// WithTemplate applies a preset edit configuration.
func WithTemplate(t TemplateType) Option {
	return func(o *Options) {
		switch t {
		case TemplateBlurred:
			o.Background = BlurredVideo
			o.ForegroundSize = Size{1080, 607}
		case TemplateBlack:
			o.Background = BlackScreen
			o.ForegroundSize = Size{1080, 607}
		case TemplateImage:
			o.Background = StaticImage
			o.ForegroundSize = Size{1080, 607}
		}
	}
}

// WithTitle sets an overlay title text on the output video.
func WithTitle(title string) Option {
	return func(o *Options) {
		o.Title = title
	}
}
