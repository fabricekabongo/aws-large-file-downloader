package app

import (
	"context"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"go.opentelemetry.io/otel"

	"github.com/example/aws-large-file-downloader/internal/tui"
)

func Run(ctx context.Context, logger *slog.Logger) error {
	tr := otel.Tracer("cli")
	ctx, span := tr.Start(ctx, "app.run")
	defer span.End()

	logger.InfoContext(ctx, "starting tui")
	p := tea.NewProgram(tui.NewModel(), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	_, err := p.Run()
	if err != nil {
		logger.ErrorContext(ctx, "tui failed", slog.String("error", err.Error()))
		return err
	}
	logger.InfoContext(ctx, "tui completed")
	return nil
}
