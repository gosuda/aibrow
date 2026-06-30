package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/store"
	ff "github.com/u2takey/ffmpeg-go"
)

func setupToolTest(t *testing.T) (*store.Store, *ffmpeg.Client, string) {
	t.Helper()
	dir := t.TempDir()
	s := store.NewStore()
	client := ffmpeg.NewClient()
	return s, client, dir
}

func createImage(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=red:s=64x64", "-frames:v", "1", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v\n%s", err, out)
	}
}

func TestAnalyzeImageTool(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	r := NewDefaultRegistry(s, client, dir)
	out, err := r.Execute(context.Background(), analyzeImageName, json.RawMessage(`{"image_id":"`+record.ID+`"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	res, ok := out.(AnalyzeImageResult)
	if !ok {
		t.Fatalf("unexpected result type %T", out)
	}
	if res.Analysis.Width != 64 {
		t.Errorf("width = %d, want 64", res.Analysis.Width)
	}
}

func TestAnalyzeImageNotFound(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), analyzeImageName, json.RawMessage(`{"image_id":"missing"}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAnalyzeImageParseError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), analyzeImageName, json.RawMessage(`not json`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAnalyzeImageClientError(t *testing.T) {
	s, _, dir := setupToolTest(t)
	client := ffmpeg.NewClient(ffmpeg.WithExecutor(&errorExecutor{}))
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), analyzeImageName, json.RawMessage(`{"image_id":"`+record.ID+`"}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdjustImageTool(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	r := NewDefaultRegistry(s, client, dir)
	input := AdjustImageInput{ImageID: record.ID, Params: ffmpeg.DefaultAdjustParams()}
	input.Params.Brightness = 0.1
	raw, _ := json.Marshal(input)
	out, err := r.Execute(context.Background(), adjustImageName, raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	res := out.(AdjustImageResult)
	if _, err := os.Stat(res.ProcessedPath); os.IsNotExist(err) {
		t.Error("expected processed file")
	}
}

func TestAdjustImageNotFound(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), adjustImageName, json.RawMessage(`{"image_id":"missing","params":{}}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestConvertFormatTool(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	r := NewDefaultRegistry(s, client, dir)
	out, err := r.Execute(context.Background(), convertFormatName, json.RawMessage(`{"image_id":"`+record.ID+`","format":"jpg","quality":"5"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	res := out.(ConvertFormatResult)
	if _, err := os.Stat(res.Path); os.IsNotExist(err) {
		t.Error("expected converted file")
	}
}

func TestConvertFormatNotFound(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), convertFormatName, json.RawMessage(`{"image_id":"missing","format":"png"}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestExportExcelTool(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	r := NewDefaultRegistry(s, client, dir)
	out, err := r.Execute(context.Background(), exportExcelName, json.RawMessage(`{"image_id":"`+record.ID+`"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	res := out.(ExportExcelResult)
	if _, err := os.Stat(res.Path); os.IsNotExist(err) {
		t.Error("expected excel file")
	}
}

func TestExportExcelNotFound(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), exportExcelName, json.RawMessage(`{"image_id":"missing"}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdjustImageParseError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), adjustImageName, json.RawMessage(`not json`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdjustImageClientError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	orig := ffmpeg.RunFFmpegFn
	ffmpeg.RunFFmpegFn = func(binary string, args ...string) error { return errors.New("adjust failed") }
	defer func() { ffmpeg.RunFFmpegFn = orig }()

	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), adjustImageName, json.RawMessage(`{"image_id":"`+record.ID+`","params":{"brightness":0.1}}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestConvertFormatParseError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), convertFormatName, json.RawMessage(`not json`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestConvertFormatClientError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	orig := ffmpeg.RunFFmpegFn
	ffmpeg.RunFFmpegFn = func(binary string, args ...string) error { return errors.New("convert failed") }
	defer func() { ffmpeg.RunFFmpegFn = orig }()

	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), convertFormatName, json.RawMessage(`{"image_id":"`+record.ID+`","format":"png"}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestConvertFormatUsesProcessedPath(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})
	record.ProcessedPath = filepath.Join(dir, "processed.png")
	_ = s.Update(record)

	var sourceArg string
	orig := ffmpeg.RunFFmpegFn
	ffmpeg.RunFFmpegFn = func(binary string, args ...string) error {
		for i, a := range args {
			if a == "-i" && i+1 < len(args) {
				sourceArg = args[i+1]
			}
		}
		return nil
	}
	defer func() { ffmpeg.RunFFmpegFn = orig }()

	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), convertFormatName, json.RawMessage(`{"image_id":"`+record.ID+`","format":"png"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sourceArg != record.ProcessedPath {
		t.Errorf("source = %q, want %q", sourceArg, record.ProcessedPath)
	}
}

func TestExportExcelParseError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), exportExcelName, json.RawMessage(`not json`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestExportExcelWriteError(t *testing.T) {
	s, client, dir := setupToolTest(t)
	img := filepath.Join(dir, "in.png")
	createImage(t, img)
	record := s.Save(store.ImageRecord{OriginalPath: img})

	// Make outputDir a file so Excel save fails.
	dir = filepath.Join(dir, "file")
	_ = os.WriteFile(dir, []byte("x"), 0o644)

	r := NewDefaultRegistry(s, client, dir)
	_, err := r.Execute(context.Background(), exportExcelName, json.RawMessage(`{"image_id":"`+record.ID+`"}`))
	if err == nil {
		t.Error("expected error")
	}
}

type errorExecutor struct{}

func (e *errorExecutor) Probe(fileName string, kwargs ...ff.KwArgs) (string, error) {
	return "", errors.New("probe failed")
}

func (e *errorExecutor) Run(stream *ff.Stream, options ...ff.CompilationOption) error {
	return nil
}
