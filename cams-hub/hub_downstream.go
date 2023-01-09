package main

import (
	"github.com/jfsmig/cams/api/pb"
	"github.com/jfsmig/cams/utils"
	"github.com/juju/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (hub *grpcHub) Control(stream pb.Downstream_ControlServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.InvalidArgument, "missing metadata")
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}
	user := md.Get(utils.KeyUser)[0]

	if hub.agents.Has(AgentID(user)) {
		err := status.Error(codes.AlreadyExists, "user agents already running")
		utils.Logger.Warn().Str("user", user).Str("action", "check").Err(err).Msg("hub ctrl")
		return err
	}

	utils.Logger.Trace().Str("user", user).Str("action", "start").Msg("hub ctrl")

	agent := NewAgentTwin(AgentID(user), stream)
	hub.agents.Add(agent)

	// wait for commands from outside, to propagate to the agents
	for running := true; running; {
		select {
		case <-stream.Context().Done():
			utils.Logger.Info().Str("user", user).Str("action", "shut").Msg("hub ctrl")
			running = false
		case cmd := <-agent.requests:
			switch cmd.cmdType {
			case CtrlCommandType_Play: // Play a stream
				stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_PLAY, StreamID: cmd.streamID})
			case CtrlCommandType_Stop: // Stop a stream
				stream.Send(&pb.DownstreamControlRequest{Command: pb.DownstreamCommandType_DOWNSTREAM_COMMAND_TYPE_STOP, StreamID: cmd.streamID})
			case CtrlCommandType_Exit: // abort the
				running = false
			}
		case done := <-agent.terminations:
			agent.medias.Remove(done)
			utils.Logger.Info().Str("user", user).Str("stream", string(done)).Msg("hub ctrl")
		}
	}

	// Close all the media streams
	for _, media := range agent.medias {
		media.Exit()
	}
	agent.mediasWaitGroup.Wait()

	// Close the command channel
	close(agent.requests)

	// Unregister the AgentTwin
	hub.agents.Remove(AgentID(user))

	return nil
}

// An upload is starting.
func (hub *grpcHub) MediaUpload(stream pb.Downstream_MediaUploadServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		err := status.Error(codes.InvalidArgument, "missing metadata")
		utils.Logger.Warn().Str("action", "check").Err(err).Msg("hub media")
		return err
	}
	userId := md.Get(utils.KeyUser)[0]
	streamId := md.Get(utils.KeyStream)[0]

	utils.Logger.Trace().Str("action", "starting").
		Str("user", userId).Str("cam", streamId).
		Msg("hub media")

	// Ensure the digital twin of the agents exists (it has been created at the provisionning step.
	// and that we create the ownly media upstream for that digital twin.
	agent, ok := hub.agents.Get(AgentID(userId))
	if !ok {
		return status.Error(codes.NotFound, "agent unknown")
	}

	upstream, err := agent.CreateStream(StreamID(streamId))
	if err != nil {
		return status.Error(codes.AlreadyExists, "stream found")
	}

	receiverDone := make(chan error, 0)
	mediaSource := make(chan *pb.DownstreamMediaFrame)

	go hub.recvMedia(stream, receiverDone, mediaSource)
	go hub.handleMedia(agent, upstream, receiverDone, mediaSource)

	for running := true; running; {
		select {
		case <-stream.Context().Done():
			utils.Logger.Warn().Str("action", "done").Msg("hub")
			running = false
		case req := <-upstream.requests:
			utils.Logger.Warn().Str("action", "req").Msg("hub")
			switch req {
			case MediaCommand_Exit:
				running = false
			default:
				utils.Logger.Warn().Msg("Unexpected command")
				running = false
			}
		}
	}

	_ = stream.SendMsg(&pb.None{})
	return nil
}

func (hub *grpcHub) recvMedia(stream pb.Downstream_MediaUploadServer, done chan error, media chan *pb.DownstreamMediaFrame) {
	defer close(done)
	defer close(media)
	for {
		if msg, err := stream.Recv(); err != nil {
			utils.Logger.Warn().Str("action", "error").Msg("hub")
			done <- err
			return
		} else {
			utils.Logger.Warn().Str("action", "frame").Msg("hub")
			media <- msg
		}
	}
}

func (hub *grpcHub) handleMedia(agent *AgentTwin, upstream *agentStream, done chan error, media chan *pb.DownstreamMediaFrame) {
	defer func() { agent.terminations <- upstream.PK() }()

	for {
		select {
		case frame, ok := <-media:
			if !ok {
				utils.Logger.Warn().Str("action", "close").Msg("hub media")
				return
			} else {
				utils.Logger.Warn().Str("action", "recv").Msg("hub media")
				switch frame.Type {
				case pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtp").Msg("hub media")
				case pb.DownstreamMediaFrameType_DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
					// TODO(jfs): push the frame to its listeners
					utils.Logger.Info().Str("proto", "rtcp").Msg("hub media")
				default:
					done <- errors.New("unexpected control message type")
				}
			}
		case err, _ := <-done:
			utils.Logger.Warn().Str("action", "abort").Err(err).Msg("hub media")
			return
		}
	}
}
