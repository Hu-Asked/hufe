package fileops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func ValidateTrashDestination(source, trashDirectory string) (string, error) {
	if trashDirectory == "" {
		return "", errors.New("HUFE_TRASH_DIR is not set")
	}
	if !filepath.IsAbs(trashDirectory) {
		return "", errors.New("HUFE_TRASH_DIR must be an absolute path")
	}

	source, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("resolve item: %w", err)
	}
	trashDirectory = filepath.Clean(trashDirectory)
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return "", fmt.Errorf("read item: %w", err)
	}
	info, err := os.Stat(trashDirectory)
	if err != nil {
		return "", fmt.Errorf("read trash directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("trash destination is not a directory: %s", trashDirectory)
	}
	resolvedTrash, err := filepath.EvalSymlinks(trashDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve trash directory: %w", err)
	}
	resolvedSource := source
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		resolvedParent, resolveErr := filepath.EvalSymlinks(filepath.Dir(source))
		if resolveErr != nil {
			return "", fmt.Errorf("resolve item parent: %w", resolveErr)
		}
		resolvedSource = filepath.Join(resolvedParent, filepath.Base(source))
	} else {
		resolvedSource, err = filepath.EvalSymlinks(source)
		if err != nil {
			return "", fmt.Errorf("resolve item: %w", err)
		}
	}
	if pathContains(resolvedTrash, resolvedSource) {
		return "", errors.New("items already inside HUFE_TRASH_DIR cannot be deleted")
	}
	if pathContains(resolvedSource, resolvedTrash) {
		return "", errors.New("HUFE_TRASH_DIR cannot be inside the item being deleted")
	}
	return trashDirectory, nil
}

func MoveToTrash(ctx context.Context, source, trashDirectory string, report Reporter) (Result, error) {
	return moveToTrash(ctx, source, trashDirectory, report, os.Rename, time.Now())
}

func moveToTrash(ctx context.Context, source, trashDirectory string, report Reporter, rename func(string, string) error, now time.Time) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	validatedTrash, err := ValidateTrashDestination(source, trashDirectory)
	if err != nil {
		return Result{}, err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return Result{}, fmt.Errorf("resolve item: %w", err)
	}
	targetName, err := availableTrashName(validatedTrash, filepath.Base(source), now)
	if err != nil {
		return Result{}, err
	}
	target := filepath.Join(validatedTrash, targetName)

	if err := rename(source, target); err == nil {
		return Result{Target: target}, nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return Result{}, fmt.Errorf("move item to trash: %w", err)
	}

	result, err := copyTo(ctx, source, validatedTrash, targetName, ".hufe-trash-", report)
	if err != nil {
		return Result{}, fmt.Errorf("copy item to trash: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := os.RemoveAll(source); err != nil {
		return result, fmt.Errorf("trash copy completed but original could not be removed: %w", err)
	}
	return result, nil
}

func availableTrashName(directory, baseName string, now time.Time) (string, error) {
	prefix := fmt.Sprintf("%s.deleted-%s", baseName, now.Format("20060102-150405"))
	for counter := 1; ; counter++ {
		candidate := prefix
		if counter > 1 {
			candidate = fmt.Sprintf("%s-%d", prefix, counter)
		}
		_, err := os.Lstat(filepath.Join(directory, candidate))
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("check trash destination: %w", err)
		}
	}
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)))
}
