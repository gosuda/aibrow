package ffmpeg

import (
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// Executor abstracts ffmpeg-go operations so tests can substitute them.
type Executor interface {
	Probe(fileName string, kwargs ...ffmpeg.KwArgs) (string, error)
	Run(stream *ffmpeg.Stream, options ...ffmpeg.CompilationOption) error
}

// defaultExecutor is the production Executor backed by ffmpeg-go.
type defaultExecutor struct{}

func (defaultExecutor) Probe(fileName string, kwargs ...ffmpeg.KwArgs) (string, error) {
	return ffmpeg.Probe(fileName, kwargs...)
}

func (defaultExecutor) Run(stream *ffmpeg.Stream, options ...ffmpeg.CompilationOption) error {
	return stream.Run(options...)
}
