// Copyright 2016 The Vuvuzela Authors. All rights reserved.
// Use of this source code is governed by the GNU AGPL
// license that can be found in the LICENSE file.

package main

import (
	"regexp"

	"vuvuzela.io/alpenhorn"
	"vuvuzela.io/alpenhorn/config"
	"vuvuzela.io/alpenhorn/log"
)

func (gc *GuiClient) Error(err error) {
	if *debug {
		log.Error(err)
		return
	}
	// Ignore some errors by default and do it the hacky way since
	// we don't have a great plan for errors anyway.
	matched, _ := regexp.MatchString("round [0-9]+ not configured", err.Error())
	if matched {
		return
	}
	log.Error(err)
}

func (gc *GuiClient) DebugError(err error) {
	if *debug {
		log.Error(err)
		return
	}
}

func (gc *GuiClient) ConfirmedFriend(f *alpenhorn.Friend) {
	gc.WarnfSync("Confirmed friend: %s\n", f.Username)
}

func (gc *GuiClient) SentFriendRequest(r *alpenhorn.OutgoingFriendRequest) {
	gc.WarnfSync("Sent friend request: %s\n", r.Username)
}

func (gc *GuiClient) ReceivedFriendRequest(r *alpenhorn.IncomingFriendRequest) {
	gc.WarnfSync("Received friend request: %s\n", r.Username)
	gc.WarnfSync("Type `/approve %s` to approve the friend request.\n", r.Username)
	notify("Friend request from %s", r.Username)
}

func (gc *GuiClient) UnexpectedSigningKey(in *alpenhorn.IncomingFriendRequest, out *alpenhorn.OutgoingFriendRequest) {
	gc.WarnfSync("Unexpected signing key: %s\n", in.Username)
}

func (gc *GuiClient) SendingCall(call *alpenhorn.OutgoingCall) {
	convo := gc.getOrCreateConvo(call.Username)
	round, err := gc.convoClient.LatestRound()
	if err != nil {
		convo.WarnfSync("Error calling %s: failed to fetch latest convo round: %s\n", err)
		return
	}
	convo.WarnfSync("Calling %s ...\n", call.Username)
	epochStart, intent := stdRoundSyncer.outgoingCallConvoRound(round)

	call.UpdateIntent(intent)

	wheel := &keywheelStart{
		sessionKey: call.SessionKey(),
		convoRound: epochStart,
	}
	if !gc.activateConvo(convo, wheel) {
		convo.Lock()
		convo.pendingCall = wheel
		convo.Unlock()
		convo.WarnfSync("Too many active conversations! Hang up another convo and type /answer to answer the call.\n")
	}
}

func (gc *GuiClient) ReceivedCall(call *alpenhorn.IncomingCall) {
	convo := gc.getOrCreateConvo(call.Username)
	convo.WarnfSync("Received call: %s\n", call.Username)
	notify("Call from %s", call.Username)

	round, err := gc.convoClient.LatestRound()
	if err != nil {
		convo.WarnfSync("Error activating convo: failed to fetch latest convo round: %s\n", err)
		return
	}
	wheel := &keywheelStart{
		sessionKey: call.SessionKey,
		convoRound: stdRoundSyncer.incomingCallConvoRound(round, call.Intent),
	}
	if !gc.activateConvo(convo, wheel) {
		convo.Lock()
		convo.pendingCall = wheel
		convo.Unlock()
		convo.WarnfSync("Too many active conversations! Hang up another convo and type /answer to answer the call.\n")
	}
}

func (gc *GuiClient) NewConfig(chain []*config.SignedConfig) {
	// TODO we should let the user know the differences between versions
	prev := chain[len(chain)-1]
	next := chain[0]
	gc.WarnfSync("New %q config: %s -> %s\n", prev.Service, prev.Hash(), next.Hash())
	notify("New %s config", prev.Service)
}

func (gc *GuiClient) GlobalAnnouncement(message string) {
	gc.WarnfSync("Global Announcement: %s\n", message)
	notify("Global Announcement: %s", message)
}
