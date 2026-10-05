package main

/*
Methods supported:
	Subscribe
	Publish

Subscribe:
	takes topic name, client ip:port
	adds ip:port to list of subscribers to topic

Publish:
	takes topic name, message
	for all peers under topic:
		send request {topic, message} to all peers under topic
		if failed, retry?
*/

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type Peer struct {
	ip   string
	port int
}

func (p *Peer) Url() string {
	return fmt.Sprintf("http://%s:%d", p.ip, p.port)
}

type PubSubHandler struct {
	subs map[string][]*Peer
	mu   sync.Mutex
}

func NewPubSubHandler() *PubSubHandler {
	return &PubSubHandler{
		subs: make(map[string][]*Peer),
		mu:   sync.Mutex{},
	}
}

func (p *PubSubHandler) GetExistingPeer(topic string, peerIp string, peerPort int) *Peer {
	for _, peer := range p.subs[topic] {
		if (peer.ip == peerIp) && (peer.port == peerPort) {
			return peer
		}
	}

	return nil
}

func (p *PubSubHandler) Subscribe(topic string, peerIp string, peerPort int) (*Peer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if existingPeer := p.GetExistingPeer(topic, peerIp, peerPort); existingPeer != nil {
		return existingPeer, nil
	}

	newPeer := &Peer{
		ip:   peerIp,
		port: peerPort,
	}
	if _, ok := p.subs[topic]; !ok {
		p.subs[topic] = []*Peer{}
	}
	p.subs[topic] = append(p.subs[topic], newPeer)
	return newPeer, nil
}

func (p *PubSubHandler) Unsubscribe(topic string, peerIp string, peerPort int) *Peer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peerList, ok := p.subs[topic]; ok {
		for idx, peer := range peerList {
			if peer.ip == peerIp && peer.port == peerPort {
				p.subs[topic] = append(peerList[:idx], peerList[idx+1:]...)
				return peer
			}
		}
	}
	return nil
}

func (p *PubSubHandler) WriteData(peer *Peer, topic string, message []byte) {
	url := peer.Url() + "?topic=" + topic
	resp, err := http.Post(url, "application/octet-stream", bytes.NewReader(message))
	if err != nil {
		fmt.Printf("topic: %s, peer: %v failed with error %v\n", topic, peer, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("topic: %s, peer: %v failed with status code %d\n", topic, peer, resp.StatusCode)
		return
	}
}

func (p *PubSubHandler) Publish(topic string, message []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peerList, ok := p.subs[topic]; !ok {
		return fmt.Errorf("Publish: could not find topic %s", topic)
	} else {
		for _, peer := range peerList {
			go p.WriteData(peer, topic, message)
		}
	}
	return nil
}

func (p *PubSubHandler) SubscribeHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vals := r.URL.Query()
		topic := vals.Get("topic")
		if topic == "" {
			w.WriteHeader(400)
			w.Write([]byte("no topic found"))
			return
		}
		peerIp := vals.Get("peerIp")
		if peerIp == "" {
			w.WriteHeader(400)
			w.Write([]byte("no peerIp found"))
			return
		}
		peerPort := vals.Get("peerPort")
		if peerPort == "" {
			w.WriteHeader(400)
			w.Write([]byte("no peerIp found"))
			return
		}
		peerPortInt, err := strconv.Atoi(peerPort)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("could not get proper port %v", err)))
			return
		}
		peer, _ := p.Subscribe(topic, peerIp, peerPortInt)
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("subscribed %v to topic %s", peer, topic)))
		fmt.Printf("new handler %v\n", p)
	}
}

func (p *PubSubHandler) PublishHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vals := r.URL.Query()
		topic := vals.Get("topic")
		if topic == "" {
			w.WriteHeader(400)
			w.Write([]byte("no topic found"))
			return
		}
		fullBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("could not get request body %v", err)))
			return
		}
		err = p.Publish(topic, fullBody)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(fmt.Sprintf("could not get request body %v", err)))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("published to topic %s", topic)))
	}
}

func main() {
	handler := NewPubSubHandler()
	address := flag.String("address", "0.0.0.0", "--address defines the address to bind the server to.")
	port := flag.Int("port", 8080, "--port defines the port to bind the server to.")

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, this is your Go HTTP server!")
	})

	http.HandleFunc("/subscribe", handler.SubscribeHandler())
	http.HandleFunc("/publish", handler.PublishHandler())

	listenAddr := fmt.Sprintf("%s:%d", *address, *port)
	fmt.Printf("Server listening on %s\n", listenAddr)
	http.ListenAndServe(listenAddr, nil)
}
