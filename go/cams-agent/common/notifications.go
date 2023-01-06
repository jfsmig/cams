package common

type CameraState uint32

const (
	CameraState_Online CameraState = iota
	CameraState_Offline
)

type CameraObserver interface {
	PK() string
	UpdateCameraState(camId string, state CameraState)
}

type StreamExpectation string

const (
	UpstreamAgent_ExpectPlay  StreamExpectation = "play"
	UpstreamAgent_ExpectPause                   = "pause"
)

type StreamExpectancyObserver interface {
	PK() string
	UpdateStreamExpectation(camID string, cmd StreamExpectation)
}
