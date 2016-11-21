package main

import (
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"time"
)

type ETCDWatchCallback func(*etcd.Response, error)

type ETCDCTL struct {
	client etcd.Client
	kapi   etcd.KeysAPI
	ctx    context.Context
	cfg    etcd.Config
}

func NewETCDCTL(endpoints []string, timeout time.Duration, username, password string) (*ETCDCTL, error) {
	etcdctl := new(ETCDCTL)
	cfg := etcd.Config{
		Endpoints:               endpoints,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: timeout,
		Username:                username,
		Password:                password,
	}
	etcdctl.cfg = cfg
	client, err := etcd.New(cfg)
	if err != nil {
		return nil, err
	}
	etcdctl.client = client
	etcdctl.kapi = etcd.NewKeysAPI(client)
	etcdctl.ctx = context.Background()

	return etcdctl, nil
}

func (e *ETCDCTL) Get(key string) (*etcd.Response, error) {
	options := &etcd.GetOptions{
		Quorum: true,
	}
	e.client.Sync(e.ctx)
	return e.kapi.Get(e.ctx, key, options)
}

func (e *ETCDCTL) GetWithRecursive(key string) (*etcd.Response, error) {
	options := &etcd.GetOptions{
		Recursive: true,
	}
	e.client.Sync(e.ctx)
	return e.kapi.Get(e.ctx, key, options)
}

func (e *ETCDCTL) Set(key, val string, dir bool) (*etcd.Response, error) {
	options := &etcd.SetOptions{
		Dir: dir,
	}
	return e.kapi.Set(e.ctx, key, val, options)
}

func (e *ETCDCTL) SetWithTTL(key, val string, dir bool, ttl time.Duration) (*etcd.Response, error) {
	options := &etcd.SetOptions{
		TTL: ttl,
		Dir: dir,
	}
	return e.kapi.Set(e.ctx, key, val, options)
}

func (e *ETCDCTL) Watch(key string, recursive bool) (*etcd.Response, error) {
	options := &etcd.WatcherOptions{
		Recursive: recursive,
	}
	e.client.Sync(e.ctx)
	watcher := e.kapi.Watcher(key, options)
	return watcher.Next(e.ctx)
}

func (e *ETCDCTL) WatchForever(key string, callback ETCDWatchCallback, recursive bool) {
	go func() {
		options := &etcd.WatcherOptions{
			Recursive: recursive,
		}
		for {
			e.client.Sync(e.ctx)
			watcher := e.kapi.Watcher(key, options)
			resp, err := watcher.Next(e.ctx)
			go callback(resp, err)
		}
	}()
}

func (e *ETCDCTL) Delete(key string, recursive bool) (*etcd.Response, error) {
	options := &etcd.DeleteOptions{
		Recursive: recursive,
	}
	return e.kapi.Delete(e.ctx, key, options)
}
