package main

type ifRow2 struct {
	Luid                                                                            uint64
	Index                                                                           uint32
	Guid                                                                            [16]byte
	Alias, Description                                                              [257]uint16
	PhysicalAddressLength                                                           uint32
	PhysicalAddress, PermanentPhysicalAddress                                       [32]byte
	Mtu, Type, TunnelType, MediaType, PhysicalMediumType, AccessType, DirectionType uint32
	Flags                                                                           uint8
	OperStatus, AdminStatus, MediaConnectState                                      uint32
	NetworkGuid                                                                     [16]byte
	ConnectionType                                                                  uint32
	TransmitLinkSpeed, ReceiveLinkSpeed                                             uint64
	InOctets, InUcastPkts, InNUcastPkts, InDiscards, InErrors, InUnknownProtos      uint64
	InUcastOctets, InMulticastOctets, InBroadcastOctets                             uint64
	OutOctets, OutUcastPkts, OutNUcastPkts, OutDiscards, OutErrors                  uint64
	OutUcastOctets, OutMulticastOctets, OutBroadcastOctets, OutQLen                 uint64
}

type byteCounter struct{ rx, tx uint64 }

func counterDelta(previous, next map[uint64]byteCounter) (uint64, uint64) {
	var rx, tx uint64
	for id, value := range next {
		if old, ok := previous[id]; ok && value.rx >= old.rx && value.tx >= old.tx {
			rx += value.rx - old.rx
			tx += value.tx - old.tx
		}
	}
	return rx, tx
}
