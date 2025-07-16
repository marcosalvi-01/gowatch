package stats

import (
	"context"
	_ "embed"
	"gowatch/db"
	"slices"

	"github.com/a-h/templ"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type Stats struct {
	query *db.Queries
}

func NewStats(query *db.Queries) *Stats {
	return &Stats{
		query: query,
	}
}

func (s *Stats) MostWatchedMovie() (templ.Component, error) {
	movies, err := s.query.GetMostWatchedMovies(context.Background())
	if err != nil {
		return nil, err
	}
	// Check if we have data
	if len(movies) == 0 {
		return nil, nil
	}

	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithGridOpts(opts.Grid{
			Left: "100px",
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Theme:           types.ThemeChalk,
			Width:           "100%",
			Height:          "500px",
			BackgroundColor: "transparent",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Most Watched Movies",
			Subtitle: "Top movies by view count",
			Left:     "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "value",
			Name: "Views",
			AxisLabel: &opts.AxisLabel{
				Show: opts.Bool(false),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "category",
			AxisLabel: &opts.AxisLabel{
				Show:     opts.Bool(true),
				Color:    "white",
				FontSize: 14,
				Width:    100,
				Overflow: "truncate",
				Ellipsis: "...",
			},
		}),
	)

	bar.XYReversal()

	limit := min(len(movies), 10)
	var movieTitles []string
	var viewCounts []opts.BarData

	for i := range limit {
		movie := movies[i]
		movieTitles = append(movieTitles, movie.Title)
		viewCounts = append(viewCounts, opts.BarData{
			Value: movie.ViewCount,
			Name:  movie.Title,
		})
	}

	slices.Reverse(movieTitles)
	slices.Reverse(viewCounts)

	bar.SetXAxis(movieTitles).
		AddSeries("Views", viewCounts).
		SetSeriesOptions(
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "#3b82f6",
			}),
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true),
				Position: "right",
				Color:    "white", // Text color
			}))

	return ConvertChartToTemplComponent(bar), nil
}
