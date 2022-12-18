// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"encoding/json"
	smedia "github.com/use-go/onvif/sdk/media"
	"os"

	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	"github.com/use-go/onvif/media"
	sdev "github.com/use-go/onvif/sdk/device"

	"github.com/jfsmig/cams/go/utils"
)

type DeviceOutput struct {
	AccessPolicy             *device.GetAccessPolicyResponse
	CACertificates           *device.GetCACertificatesResponse
	Capabilities             *device.GetCapabilitiesResponse
	CertificateInformation   *device.GetCertificateInformationResponse
	Certificates             *device.GetCertificatesResponse
	CertificatesStatus       *device.GetCertificatesStatusResponse
	ClientCertificateMode    *device.GetClientCertificateModeResponse
	DeviceInformation        *device.GetDeviceInformationResponse
	DiscoveryMode            *device.GetDiscoveryModeResponse
	DNS                      *device.GetDNSResponse
	Dot11Capabilities        *device.GetDot11CapabilitiesResponse
	Dot11Status              *device.GetDot11StatusResponse
	Dot1XConfiguration       *device.GetDot1XConfigurationResponse
	Dot1XConfigurations      *device.GetDot1XConfigurationsResponse
	DPAddresses              *device.GetDPAddressesResponse
	DynamicDNS               *device.GetDynamicDNSResponse
	EndpointReference        *device.GetEndpointReferenceResponse
	GeoLocation              *device.GetGeoLocationResponse
	Hostname                 *device.GetHostnameResponse
	IPAddressFilter          *device.GetIPAddressFilterResponse
	NetworkDefaultGateway    *device.GetNetworkDefaultGatewayResponse
	NetworkInterfaces        *device.GetNetworkInterfacesResponse
	NetworkProtocols         *device.GetNetworkProtocolsResponse
	NTP                      *device.GetNTPResponse
	Pkcs10Request            *device.GetPkcs10RequestResponse
	RelayOutputs             *device.GetRelayOutputsResponse
	RemoteDiscoveryMode      *device.GetRemoteDiscoveryModeResponse
	RemoteUser               *device.GetRemoteUserResponse
	Scopes                   *device.GetScopesResponse
	ServiceCapabilities      *device.GetServiceCapabilitiesResponse
	Services                 *device.GetServicesResponse
	StorageConfiguration     *device.GetStorageConfigurationResponse
	StorageConfigurations    *device.GetStorageConfigurationsResponse
	SystemBackup             *device.GetSystemBackupResponse
	SystemDateAndTime        *device.GetSystemDateAndTimeResponse
	SystemLog                *device.GetSystemLogResponse
	SystemSupportInformation *device.GetSystemSupportInformationResponse
	SystemUris               *device.GetSystemUrisResponse
	Users                    *device.GetUsersResponse
	WsdlUrl                  *device.GetWsdlUrlResponse
	ZeroConfiguration        *device.GetZeroConfigurationResponse
}

type MediaOutput struct {
	AudioDecoderConfiguration               *media.GetAudioDecoderConfigurationResponse
	AudioDecoderConfigurationOptions        *media.GetAudioDecoderConfigurationOptionsResponse
	AudioDecoderConfigurations              *media.GetAudioDecoderConfigurationsResponse
	AudioEncoderConfiguration               *media.GetAudioEncoderConfigurationResponse
	AudioEncoderConfigurationOptions        *media.GetAudioEncoderConfigurationOptionsResponse
	AudioEncoderConfigurations              *media.GetAudioEncoderConfigurationsResponse
	AudioOutputConfiguration                *media.GetAudioOutputConfigurationResponse
	AudioOutputConfigurationOptions         *media.GetAudioOutputConfigurationOptionsResponse
	AudioOutputConfigurations               *media.GetAudioOutputConfigurationsResponse
	AudioOutputs                            *media.GetAudioOutputsResponse
	AudioSourceConfiguration                *media.GetAudioSourceConfigurationResponse
	AudioSourceConfigurationOptions         *media.GetAudioSourceConfigurationOptionsResponse
	AudioSourceConfigurations               *media.GetAudioSourceConfigurationsResponse
	AudioSources                            *media.GetAudioSourcesResponse
	CompatibleAudioDecoderConfigurations    *media.GetCompatibleAudioDecoderConfigurationsResponse
	CompatibleAudioEncoderConfigurations    *media.GetCompatibleAudioEncoderConfigurationsResponse
	CompatibleAudioOutputConfigurations     *media.GetCompatibleAudioOutputConfigurationsResponse
	CompatibleAudioSourceConfigurations     *media.GetCompatibleAudioSourceConfigurationsResponse
	CompatibleMetadataConfigurations        *media.GetCompatibleMetadataConfigurationsResponse
	CompatibleVideoAnalyticsConfigurations  *media.GetCompatibleVideoAnalyticsConfigurationsResponse
	CompatibleVideoEncoderConfigurations    *media.GetCompatibleVideoEncoderConfigurationsResponse
	CompatibleVideoSourceConfigurations     *media.GetCompatibleVideoSourceConfigurationsResponse
	GuaranteedNumberOfVideoEncoderInstances *media.GetGuaranteedNumberOfVideoEncoderInstancesResponse
	MetadataConfiguration                   *media.GetMetadataConfigurationResponse
	MetadataConfigurationOptions            *media.GetMetadataConfigurationOptionsResponse
	MetadataConfigurations                  *media.GetMetadataConfigurationsResponse
	OSD                                     *media.GetOSDResponse
	OSDOptions                              *media.GetOSDOptionsResponse
	OSDs                                    *media.GetOSDsResponse
	Profile                                 *media.GetProfileResponse
	Profiles                                *media.GetProfilesResponse
	ServiceCapabilities                     *media.GetServiceCapabilitiesResponse
	SnapshotUri                             *media.GetSnapshotUriResponse
	StreamUri                               *media.GetStreamUriResponse
	VideoAnalyticsConfiguration             *media.GetVideoAnalyticsConfigurationResponse
	VideoAnalyticsConfigurations            *media.GetVideoAnalyticsConfigurationsResponse
	VideoEncoderConfiguration               *media.GetVideoEncoderConfigurationResponse
	VideoEncoderConfigurationOptions        *media.GetVideoEncoderConfigurationOptionsResponse
	VideoEncoderConfigurations              *media.GetVideoEncoderConfigurationsResponse
	VideoSourceConfiguration                *media.GetVideoSourceConfigurationResponse
	VideoSourceConfigurationOptions         *media.GetVideoSourceConfigurationOptionsResponse
	VideoSourceConfigurations               *media.GetVideoSourceConfigurationsResponse
	VideoSourceModes                        *media.GetVideoSourceModesResponse
	VideoSources                            *media.GetVideoSourcesResponse
}

type OnvifOutput struct {
	Endpoint string
	Device   DeviceOutput
	Media    MediaOutput
}

func detailMedia(ctx context.Context, dev *goonvif.Device) MediaOutput {
	var out MediaOutput

	if p, err := smedia.Call_GetAudioDecoderConfiguration(ctx, dev, media.GetAudioDecoderConfiguration{}); err == nil {
		out.AudioDecoderConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioDecoderConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetAudioDecoderConfigurationOptions(ctx, dev, media.GetAudioDecoderConfigurationOptions{}); err == nil {
		out.AudioDecoderConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioDecoderConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetAudioDecoderConfigurations(ctx, dev, media.GetAudioDecoderConfigurations{}); err == nil {
		out.AudioDecoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioDecoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetAudioEncoderConfiguration(ctx, dev, media.GetAudioEncoderConfiguration{}); err == nil {
		out.AudioEncoderConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioEncoderConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetAudioEncoderConfigurationOptions(ctx, dev, media.GetAudioEncoderConfigurationOptions{}); err == nil {
		out.AudioEncoderConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioEncoderConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetAudioEncoderConfigurations(ctx, dev, media.GetAudioEncoderConfigurations{}); err == nil {
		out.AudioEncoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioEncoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetAudioOutputConfiguration(ctx, dev, media.GetAudioOutputConfiguration{}); err == nil {
		out.AudioOutputConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioOutputConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetAudioOutputConfigurationOptions(ctx, dev, media.GetAudioOutputConfigurationOptions{}); err == nil {
		out.AudioOutputConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioOutputConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetAudioOutputConfigurations(ctx, dev, media.GetAudioOutputConfigurations{}); err == nil {
		out.AudioOutputConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioOutputConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetAudioOutputs(ctx, dev, media.GetAudioOutputs{}); err == nil {
		out.AudioOutputs = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioOutputs").Msg("media")
	}
	if p, err := smedia.Call_GetAudioSourceConfiguration(ctx, dev, media.GetAudioSourceConfiguration{}); err == nil {
		out.AudioSourceConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioSourceConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetAudioSourceConfigurationOptions(ctx, dev, media.GetAudioSourceConfigurationOptions{}); err == nil {
		out.AudioSourceConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioSourceConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetAudioSourceConfigurations(ctx, dev, media.GetAudioSourceConfigurations{}); err == nil {
		out.AudioSourceConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioSourceConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetAudioSources(ctx, dev, media.GetAudioSources{}); err == nil {
		out.AudioSources = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AudioSources").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleAudioDecoderConfigurations(ctx, dev, media.GetCompatibleAudioDecoderConfigurations{}); err == nil {
		out.CompatibleAudioDecoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleAudioDecoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleAudioEncoderConfigurations(ctx, dev, media.GetCompatibleAudioEncoderConfigurations{}); err == nil {
		out.CompatibleAudioEncoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleAudioEncoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleAudioOutputConfigurations(ctx, dev, media.GetCompatibleAudioOutputConfigurations{}); err == nil {
		out.CompatibleAudioOutputConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleAudioOutputConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleAudioSourceConfigurations(ctx, dev, media.GetCompatibleAudioSourceConfigurations{}); err == nil {
		out.CompatibleAudioSourceConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleAudioSourceConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleMetadataConfigurations(ctx, dev, media.GetCompatibleMetadataConfigurations{}); err == nil {
		out.CompatibleMetadataConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleMetadataConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleVideoAnalyticsConfigurations(ctx, dev, media.GetCompatibleVideoAnalyticsConfigurations{}); err == nil {
		out.CompatibleVideoAnalyticsConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleVideoAnalyticsConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleVideoEncoderConfigurations(ctx, dev, media.GetCompatibleVideoEncoderConfigurations{}); err == nil {
		out.CompatibleVideoEncoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleVideoEncoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetCompatibleVideoSourceConfigurations(ctx, dev, media.GetCompatibleVideoSourceConfigurations{}); err == nil {
		out.CompatibleVideoSourceConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CompatibleVideoSourceConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetGuaranteedNumberOfVideoEncoderInstances(ctx, dev, media.GetGuaranteedNumberOfVideoEncoderInstances{}); err == nil {
		out.GuaranteedNumberOfVideoEncoderInstances = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "GuaranteedNumberOfVideoEncoderInstances").Msg("media")
	}
	if p, err := smedia.Call_GetMetadataConfiguration(ctx, dev, media.GetMetadataConfiguration{}); err == nil {
		out.MetadataConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "MetadataConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetMetadataConfigurationOptions(ctx, dev, media.GetMetadataConfigurationOptions{}); err == nil {
		out.MetadataConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "MetadataConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetMetadataConfigurations(ctx, dev, media.GetMetadataConfigurations{}); err == nil {
		out.MetadataConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "MetadataConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetOSD(ctx, dev, media.GetOSD{}); err == nil {
		out.OSD = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "OSD").Msg("media")
	}
	if p, err := smedia.Call_GetOSDOptions(ctx, dev, media.GetOSDOptions{}); err == nil {
		out.OSDOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "OSDOptions").Msg("media")
	}
	if p, err := smedia.Call_GetOSDs(ctx, dev, media.GetOSDs{}); err == nil {
		out.OSDs = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "OSDs").Msg("media")
	}
	if p, err := smedia.Call_GetProfile(ctx, dev, media.GetProfile{}); err == nil {
		out.Profile = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Profile").Msg("media")
	}
	if p, err := smedia.Call_GetProfiles(ctx, dev, media.GetProfiles{}); err == nil {
		out.Profiles = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Profiles").Msg("media")
	}
	if p, err := smedia.Call_GetServiceCapabilities(ctx, dev, media.GetServiceCapabilities{}); err == nil {
		out.ServiceCapabilities = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "ServiceCapabilities").Msg("media")
	}
	if p, err := smedia.Call_GetSnapshotUri(ctx, dev, media.GetSnapshotUri{}); err == nil {
		out.SnapshotUri = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SnapshotUri").Msg("media")
	}
	if p, err := smedia.Call_GetStreamUri(ctx, dev, media.GetStreamUri{}); err == nil {
		out.StreamUri = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "StreamUri").Msg("media")
	}
	if p, err := smedia.Call_GetVideoAnalyticsConfiguration(ctx, dev, media.GetVideoAnalyticsConfiguration{}); err == nil {
		out.VideoAnalyticsConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoAnalyticsConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetVideoAnalyticsConfigurations(ctx, dev, media.GetVideoAnalyticsConfigurations{}); err == nil {
		out.VideoAnalyticsConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoAnalyticsConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetVideoEncoderConfiguration(ctx, dev, media.GetVideoEncoderConfiguration{}); err == nil {
		out.VideoEncoderConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoEncoderConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetVideoEncoderConfigurationOptions(ctx, dev, media.GetVideoEncoderConfigurationOptions{}); err == nil {
		out.VideoEncoderConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoEncoderConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetVideoEncoderConfigurations(ctx, dev, media.GetVideoEncoderConfigurations{}); err == nil {
		out.VideoEncoderConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoEncoderConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetVideoSourceConfiguration(ctx, dev, media.GetVideoSourceConfiguration{}); err == nil {
		out.VideoSourceConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoSourceConfiguration").Msg("media")
	}
	if p, err := smedia.Call_GetVideoSourceConfigurationOptions(ctx, dev, media.GetVideoSourceConfigurationOptions{}); err == nil {
		out.VideoSourceConfigurationOptions = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoSourceConfigurationOptions").Msg("media")
	}
	if p, err := smedia.Call_GetVideoSourceConfigurations(ctx, dev, media.GetVideoSourceConfigurations{}); err == nil {
		out.VideoSourceConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoSourceConfigurations").Msg("media")
	}
	if p, err := smedia.Call_GetVideoSourceModes(ctx, dev, media.GetVideoSourceModes{}); err == nil {
		out.VideoSourceModes = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoSourceModes").Msg("media")
	}
	if p, err := smedia.Call_GetVideoSources(ctx, dev, media.GetVideoSources{}); err == nil {
		out.VideoSources = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "VideoSources").Msg("media")
	}
	return out
}

func detailDevice(ctx context.Context, dev *goonvif.Device) DeviceOutput {
	var out DeviceOutput
	if p, err := sdev.Call_GetAccessPolicy(ctx, dev, device.GetAccessPolicy{}); err == nil {
		out.AccessPolicy = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "AccessPolicy").Msg("device")
	}
	if p, err := sdev.Call_GetCACertificates(ctx, dev, device.GetCACertificates{}); err == nil {
		out.CACertificates = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CACertificates").Msg("device")
	}
	if p, err := sdev.Call_GetCapabilities(ctx, dev, device.GetCapabilities{}); err == nil {
		out.Capabilities = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Capabilities").Msg("device")
	}
	if p, err := sdev.Call_GetCertificateInformation(ctx, dev, device.GetCertificateInformation{}); err == nil {
		out.CertificateInformation = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CertificateInformation").Msg("device")
	}
	if p, err := sdev.Call_GetCertificates(ctx, dev, device.GetCertificates{}); err == nil {
		out.Certificates = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Certificates").Msg("device")
	}
	if p, err := sdev.Call_GetCertificatesStatus(ctx, dev, device.GetCertificatesStatus{}); err == nil {
		out.CertificatesStatus = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "CertificatesStatus").Msg("device")
	}
	if p, err := sdev.Call_GetClientCertificateMode(ctx, dev, device.GetClientCertificateMode{}); err == nil {
		out.ClientCertificateMode = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "ClientCertificateMode").Msg("device")
	}
	if p, err := sdev.Call_GetDeviceInformation(ctx, dev, device.GetDeviceInformation{}); err == nil {
		out.DeviceInformation = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "DeviceInformation").Msg("device")
	}
	if p, err := sdev.Call_GetDiscoveryMode(ctx, dev, device.GetDiscoveryMode{}); err == nil {
		out.DiscoveryMode = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "DiscoveryMode").Msg("device")
	}
	if p, err := sdev.Call_GetDNS(ctx, dev, device.GetDNS{}); err == nil {
		out.DNS = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "DNS").Msg("device")
	}
	if p, err := sdev.Call_GetDot11Capabilities(ctx, dev, device.GetDot11Capabilities{}); err == nil {
		out.Dot11Capabilities = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Dot11Capabilities").Msg("device")
	}
	if p, err := sdev.Call_GetDot11Status(ctx, dev, device.GetDot11Status{}); err == nil {
		out.Dot11Status = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Dot11Status").Msg("device")
	}
	if p, err := sdev.Call_GetDot1XConfiguration(ctx, dev, device.GetDot1XConfiguration{}); err == nil {
		out.Dot1XConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Dot1XConfiguration").Msg("device")
	}
	if p, err := sdev.Call_GetDot1XConfigurations(ctx, dev, device.GetDot1XConfigurations{}); err == nil {
		out.Dot1XConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Dot1XConfigurations").Msg("device")
	}
	if p, err := sdev.Call_GetDPAddresses(ctx, dev, device.GetDPAddresses{}); err == nil {
		out.DPAddresses = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "DPAddresses").Msg("device")
	}
	if p, err := sdev.Call_GetDynamicDNS(ctx, dev, device.GetDynamicDNS{}); err == nil {
		out.DynamicDNS = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "DynamicDNS").Msg("device")
	}
	if p, err := sdev.Call_GetEndpointReference(ctx, dev, device.GetEndpointReference{}); err == nil {
		out.EndpointReference = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "EndpointReference").Msg("device")
	}
	if p, err := sdev.Call_GetGeoLocation(ctx, dev, device.GetGeoLocation{}); err == nil {
		out.GeoLocation = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "GeoLocation").Msg("device")
	}
	if p, err := sdev.Call_GetHostname(ctx, dev, device.GetHostname{}); err == nil {
		out.Hostname = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Hostname").Msg("device")
	}
	if p, err := sdev.Call_GetIPAddressFilter(ctx, dev, device.GetIPAddressFilter{}); err == nil {
		out.IPAddressFilter = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "IPAddressFilter").Msg("device")
	}
	if p, err := sdev.Call_GetNetworkDefaultGateway(ctx, dev, device.GetNetworkDefaultGateway{}); err == nil {
		out.NetworkDefaultGateway = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "NetworkDefaultGateway").Msg("device")
	}
	if p, err := sdev.Call_GetNetworkInterfaces(ctx, dev, device.GetNetworkInterfaces{}); err == nil {
		out.NetworkInterfaces = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "NetworkInterfaces").Msg("device")
	}
	if p, err := sdev.Call_GetNetworkProtocols(ctx, dev, device.GetNetworkProtocols{}); err == nil {
		out.NetworkProtocols = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "NetworkProtocols").Msg("device")
	}
	if p, err := sdev.Call_GetNTP(ctx, dev, device.GetNTP{}); err == nil {
		out.NTP = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "NTP").Msg("device")
	}
	if p, err := sdev.Call_GetPkcs10Request(ctx, dev, device.GetPkcs10Request{}); err == nil {
		out.Pkcs10Request = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Pkcs10Request").Msg("device")
	}
	if p, err := sdev.Call_GetRelayOutputs(ctx, dev, device.GetRelayOutputs{}); err == nil {
		out.RelayOutputs = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "RelayOutputs").Msg("device")
	}
	if p, err := sdev.Call_GetRemoteDiscoveryMode(ctx, dev, device.GetRemoteDiscoveryMode{}); err == nil {
		out.RemoteDiscoveryMode = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "RemoteDiscoveryMode").Msg("device")
	}
	if p, err := sdev.Call_GetRemoteUser(ctx, dev, device.GetRemoteUser{}); err == nil {
		out.RemoteUser = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "RemoteUser").Msg("device")
	}
	if p, err := sdev.Call_GetScopes(ctx, dev, device.GetScopes{}); err == nil {
		out.Scopes = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Scopes").Msg("device")
	}
	if p, err := sdev.Call_GetServiceCapabilities(ctx, dev, device.GetServiceCapabilities{}); err == nil {
		out.ServiceCapabilities = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "ServiceCapabilities").Msg("device")
	}
	if p, err := sdev.Call_GetServices(ctx, dev, device.GetServices{}); err == nil {
		out.Services = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Services").Msg("device")
	}
	if p, err := sdev.Call_GetStorageConfiguration(ctx, dev, device.GetStorageConfiguration{}); err == nil {
		out.StorageConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "StorageConfiguration").Msg("device")
	}
	if p, err := sdev.Call_GetStorageConfigurations(ctx, dev, device.GetStorageConfigurations{}); err == nil {
		out.StorageConfigurations = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "StorageConfigurations").Msg("device")
	}
	if p, err := sdev.Call_GetSystemBackup(ctx, dev, device.GetSystemBackup{}); err == nil {
		out.SystemBackup = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SystemBackup").Msg("device")
	}
	if p, err := sdev.Call_GetSystemDateAndTime(ctx, dev, device.GetSystemDateAndTime{}); err == nil {
		out.SystemDateAndTime = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SystemDateAndTime").Msg("device")
	}
	if p, err := sdev.Call_GetSystemLog(ctx, dev, device.GetSystemLog{}); err == nil {
		out.SystemLog = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SystemLog").Msg("device")
	}
	if p, err := sdev.Call_GetSystemSupportInformation(ctx, dev, device.GetSystemSupportInformation{}); err == nil {
		out.SystemSupportInformation = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SystemSupportInformation").Msg("device")
	}
	if p, err := sdev.Call_GetSystemUris(ctx, dev, device.GetSystemUris{}); err == nil {
		out.SystemUris = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "SystemUris").Msg("device")
	}
	if p, err := sdev.Call_GetUsers(ctx, dev, device.GetUsers{}); err == nil {
		out.Users = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "Users").Msg("device")
	}
	if p, err := sdev.Call_GetWsdlUrl(ctx, dev, device.GetWsdlUrl{}); err == nil {
		out.WsdlUrl = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "WsdlUrl").Msg("device")
	}
	if p, err := sdev.Call_GetZeroConfiguration(ctx, dev, device.GetZeroConfiguration{}); err == nil {
		out.ZeroConfiguration = &p
	} else {
		utils.Logger.Trace().Err(err).Str("rpc", "ZeroConfiguration").Msg("device")
	}

	return out
}

func details(ctx context.Context, endpoint string) error {

	dev, err := goonvif.NewDevice(goonvif.DeviceParams{
		Xaddr:    endpoint,
		Username: "admin",
		Password: "ollyhgqo",
	})
	if err != nil {
		return errors.Trace(err)
	}

	out := OnvifOutput{}
	out.Endpoint = endpoint
	out.Device = detailDevice(ctx, dev)
	out.Media = detailMedia(ctx, dev)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}
