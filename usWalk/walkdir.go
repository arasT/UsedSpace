// Scan directory and holds informations about children files/directories
package usWalk

import (
	"os"
	"path"

	"github.com/MichaelTJones/walk"
	"github.com/orcaman/concurrent-map"

	usUI "UsedSpace/usUI"
)

// Scan the given path and holds files and directories informations
//	- givenPath: directory's path to scan
//	- cDirFilesMap: will holds informations about file/directory
//	- scanState: channel to check the scan status
func WalkGivenDir(givenPath string, cDirFilesMap cmap.ConcurrentMap, scanState chan bool) {
	walk.Walk(givenPath, func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if cDirFilesMap.Set(root, usUI.FileDirStruct{root, uint64(info.Size()), false}); info.IsDir() {
			cDirFilesMap.Set(root, usUI.FileDirStruct{root, uint64(0), true})
		}
		return nil
	})

	// Update all directories Size
	for _, k := range cDirFilesMap.Keys() {
		v, _ := cDirFilesMap.Get(k)
		currentParent := k

		// Update all parents until the given path was reached
		for currentParent != givenPath {
			currentParent = path.Dir(currentParent)

			// Update all parents directories only by using files
			if !v.(usUI.FileDirStruct).IsDir {
				currentParentObj, _ := cDirFilesMap.Get(currentParent)
				currentParentSize := currentParentObj.(usUI.FileDirStruct).Size

				cDirFilesMap.Set(currentParent, usUI.FileDirStruct{currentParent, uint64(v.(usUI.FileDirStruct).Size + currentParentSize), true})
			}
		}
	}

	// Signal that the scan is done
	scanState <- true
}
