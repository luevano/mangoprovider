package rod

import "github.com/go-rod/rod"

const ActionTypeHeader = "Mangoprovider-Action-Type"

type ActionType string

const (
	ActionManga   ActionType = "manga"
	ActionVolume  ActionType = "volume"
	ActionChapter ActionType = "chapter"
	ActionPage    ActionType = "page"
)

type Action func(*rod.Page) error
