//go:build ignore

package main

import (
	"context"
	"os"
	"runtime/trace"
)

func main() {
	if err := trace.Start(os.Stdout); err != nil {
		panic(err)
	}
	defer trace.Stop()

	ctx := context.Background()
	ctx, _ = trace.NewTask(ctx, "taskCategory")
	trace.Log(ctx, "logCategory", "logMessage")
}
