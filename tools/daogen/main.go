package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"go.uber.org/zap"
)

var logger *simplelog.SimpleZapLogger

func init() {
	zapLog, _ := zap.NewDevelopment()
	logger = simplelog.NewSimpleZapLogger(zapLog)
}

func main() {
	// 1. Define flags
	filePtr := flag.String("file", os.Getenv("GOFILE"), "Path to the Go file containing the struct definition (defaults to $GOFILE)")
	structPtr := flag.String("struct", "", "The name of the struct to process (Required)")
	tablePtr := flag.String("table", "", "The target database table name (Optional)")
	pkgPtr := flag.String("pkg", os.Getenv("GOPACKAGE"), "The package name for the generated file (defaults to $GOPACKAGE)")
	outPtr := flag.String("out", "", "Output directory (Optional, defaults to input file directory)")
	dialectPtr := flag.String("dialect", "mysql", "Database dialect (mysql, postgres) (defaults to mysql)")

	flag.Parse()

	// 2. Setup Context with parameters
	ctx := context.Background()
	logFields := []zap.Field{
		tags.String("file", *filePtr),
		tags.String("struct", *structPtr),
		tags.String("table", *tablePtr),
		tags.String("pkg", *pkgPtr),
		tags.String("out", *outPtr),
		tags.String("dialect", *dialectPtr),
	}
	ctx = context.WithValue(ctx, simplelog.SimpleLogKeyCtx, logFields)

	// 3. Validate required arguments
	if *structPtr == "" {
		logger.Error(ctx, "Missing required argument: -struct")
		flag.Usage()
		os.Exit(1)
	}

	if *filePtr == "" {
		logger.Error(ctx, "Missing required argument: -file (or run via go:generate)")
		flag.Usage()
		os.Exit(1)
	}

	// 4. Resolve paths and defaults
	absFile, err := filepath.Abs(*filePtr)
	if err != nil {
		logger.Error(ctx, "Failed to resolve absolute path", tags.Error(err))
		os.Exit(1)
	}

	dir := filepath.Dir(absFile)

	outputDir := *outPtr
	if outputDir == "" {
		outputDir = dir
	}

	packageName := *pkgPtr
	if packageName == "" {
		packageName = filepath.Base(outputDir)
	}

	snakeStruct := ToSnakeCase(*structPtr)
	outputFile := filepath.Join(outputDir, fmt.Sprintf("z_%s_dao.go", snakeStruct))

	// 5. Run the parser
	logger.Info(ctx, "Processing struct")
	var p Parser = &ASTParser{}
	info, err := p.ParseStruct(ctx, absFile, *structPtr, *tablePtr)
	if err != nil {
		logger.Error(ctx, "Failed to parse struct", tags.Error(err))
		os.Exit(1)
	}

	// 6. Run the generator
	logger.Info(ctx, "Generating DAO code", tags.String("package", packageName))
	var g Generator = &TemplateGenerator{}
	content, err := g.Generate(ctx, packageName, info, *dialectPtr)
	if err != nil {
		logger.Error(ctx, "Failed to generate code", tags.Error(err))
		os.Exit(1)
	}

	// 7. Write to file
	logger.Info(ctx, "Writing generated code", tags.String("path", outputFile))
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		logger.Error(ctx, "Failed to create output directory", tags.Error(err))
		os.Exit(1)
	}
	if err := os.WriteFile(outputFile, content, 0644); err != nil {
		logger.Error(ctx, "Failed to write file", tags.Error(err))
		os.Exit(1)
	}

	logger.Info(ctx, "Successfully generated DAO", tags.String("file", outputFile))
}
