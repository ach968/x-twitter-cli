package search

import "github.com/ach968/x-twt-cli/internal/app/models"

func cleanReaderText(value string) string { return models.CleanReaderText(value) }

func cleanOptionalText(value *string) *string { return models.CleanOptionalText(value) }
