package stats

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

type Renderable interface {
	Render(w io.Writer) error
}

func ConvertChartToTemplComponent(chart Renderable) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return chart.Render(w)
	})
}
