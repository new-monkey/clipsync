package protocol

// ModeType defines the running mode for push/pull/reverse-push
// push: client主动推送，reverse-push: 被动推送，pull: 拉取
const (
	ModePush        = "push"
	ModeReversePush = "reverse-push"
	ModePull        = "pull"
)
