package search

import "github.com/ach968/x-twitter-cli/internal/app/models"

// Compatibility aliases keep the established Search package usable while the
// shared normalized post contract is introduced. New operations use models.
type postResult = models.Post
type userRef = models.UserRef
type replyTo = models.ReplyTo
type postMetrics = models.PostMetrics
type link = models.Link

func decodePostResult(raw any) (postResult, bool) { return models.DecodePostResult(raw) }

func decodeUserRef(raw map[string]any) *userRef { return models.DecodeUserRef(raw) }
