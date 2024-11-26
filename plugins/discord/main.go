package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/julien040/anyquery/rpc"
	"github.com/julien040/anyquery/rpc/helper"
)

type discordMod struct {
	sync.RWMutex
	connections map[string]*discordgo.Session
	cache       map[string]*helper.Cache
}

func (m *discordMod) getSession(token string) (*discordgo.Session, *helper.Cache, error) {
	m.RLock()
	session, ok := m.connections[token]
	cache, ok2 := m.cache[token]
	m.RUnlock()
	if ok && ok2 {
		return session, cache, nil
	}
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, nil, err
	}
	session.ShouldReconnectOnError = true
	session.ShouldRetryOnRateLimit = true

	// Open a cache session
	md5sum, sha256sum := hashToken(token)
	cache, err = helper.NewCache(helper.NewCacheArgs{
		Paths:         []string{"discord", fmt.Sprintf("%x", md5sum)},
		MaxSize:       1 << 29,
		EncryptionKey: sha256sum,
	})
	m.Lock()
	if m.connections == nil {
		m.connections = make(map[string]*discordgo.Session)
	}
	if m.cache == nil {
		m.cache = make(map[string]*helper.Cache)
	}

	m.connections[token] = session
	m.cache[token] = cache
	m.Unlock()
	return session, cache, err
}

func main() {
	connections := &discordMod{}
	plugin := rpc.NewPlugin(connections.messagesCreator, connections.channelsCreator, connections.membersCreator,
		connections.bansCreator, connections.rolesCreator, connections.guildsCreator, connections.invitesCreator)
	plugin.Serve()
}
