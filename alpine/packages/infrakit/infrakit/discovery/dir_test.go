package discovery

import (
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func blockWhileFileExists(name string) {
	for {
		_, err := os.Stat(name)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDirDiscovery(t *testing.T) {

	dir, err := ioutil.TempDir("", "infrakit_dir_test")
	require.NoError(t, err)

	name1 := "server1"
	path1 := filepath.Join(dir, name1)
	stop1, errors1, err1 := server.StartPluginAtPath(path1, mux.NewRouter())
	require.NoError(t, err1)
	require.NotNil(t, stop1)
	require.NotNil(t, errors1)

	name2 := "server2"
	path2 := filepath.Join(dir, name2)
	stop2, errors2, err2 := server.StartPluginAtPath(path2, mux.NewRouter())
	require.NoError(t, err2)
	require.NotNil(t, stop2)
	require.NotNil(t, errors2)

	discover, err := newDirPluginDiscovery(dir)
	require.NoError(t, err)

	p, err := discover.Find(name1)
	require.NoError(t, err)
	require.NotNil(t, p)

	p, err = discover.Find(name2)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Now we stop the servers
	close(stop1)
	blockWhileFileExists(path1)

	p, err = discover.Find(name1)
	require.Error(t, err)

	p, err = discover.Find(name2)
	require.NoError(t, err)
	require.NotNil(t, p)

	close(stop2)

	blockWhileFileExists(path2)

	p, err = discover.Find(name1)
	require.Error(t, err)

	p, err = discover.Find(name2)
	require.Error(t, err)

	list, err := discover.List()
	require.NoError(t, err)
	require.Equal(t, 0, len(list))
}
