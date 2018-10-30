// Contain method to
//	- Create main components: header, tree, main table
//	- Define action on these components: node selected, node changed, updating table
package usUI

import (
	//"log"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/dustin/go-humanize"
	"github.com/gdamore/tcell"
	"github.com/orcaman/concurrent-map"
	"github.com/rivo/tview"
)

// Structure to hold file/directory informations
type FileDirStruct struct {
	FullPath string
	Size     uint64
	IsDir    bool
}

// Create the header component
func createUSMainPageHeader() *tview.Flex {
	treeTitleTable := tview.NewTextView().SetScrollable(false).SetText("Navigate").SetTextColor(tcell.ColorBlue)
	listTitleTable := tview.NewTextView().SetScrollable(false).SetText("Select").SetTextColor(tcell.ColorBlue)

	return tview.NewFlex().
		AddItem(treeTitleTable, 0, 1, false).
		AddItem(listTitleTable, 0, 1, false)
}

// Create the main page (navigation tree and contents table)
//	- mainTable: table list containing selected folder's content
//	- tree: navigation tree
func SetUpMainPage(tree, mainTable tview.Primitive) *tview.Flex {
	usMainPageHeader := createUSMainPageHeader()
	usMainPageContent := tview.NewFlex().
		AddItem(tree, 0, 1, true).
		AddItem(mainTable, 0, 1, false)

	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(usMainPageHeader, 2, 1, false).
		AddItem(usMainPageContent, 0, 1, true)
}

// Add tree node for each file/directory into the selected directory from the tree
//	- target: node representing the file/directory into the tree
//	- fullPath: file/directory's full path to set into their node reference
func AddNodes(target *tview.TreeNode, fullPath string) {
	fileDir, err := os.Open(fullPath)
	if err != nil {
		//log.Fatalln("Failed to open directory: ", FullPath, ".\t Error: ", err)
		panic("Failed to open directory: " + fullPath)
	}
	defer fileDir.Close()

	// Read all direct children files and directories from the given path and create their correponding node into the tree
	fileDirList, _ := fileDir.Readdirnames(0)
	for _, fileDirName := range fileDirList {
		crtFullPath := filepath.Join(fullPath, fileDirName)
		fileDirDescription, _ := os.Lstat(crtFullPath)
		crtSize := uint64(fileDirDescription.Size())

		// Size from LStat for directory is wrong
		if fileDirDescription.IsDir() {
			crtSize = uint64(0)
		}

		// Create the node of each file/directory, set directory selectable
		crtNode := tview.NewTreeNode(fileDirName).
			SetReference(FileDirStruct{crtFullPath, crtSize, fileDirDescription.IsDir()}).
			SetSelectable(fileDirDescription.IsDir())

		// Directories are colored into green
		if fileDirDescription.IsDir() {
			crtNode.SetColor(tcell.ColorGreen).SetExpanded(false)
		}

		target.AddChild(crtNode)
	}
}

// Update table containing detailed list of files and directories children of the selected directory from the tree
//	- tree: navigation tree
//	- mainTable: table list containing selected folder's content
//	- pages: holds all pages for this application
//	- fileDirData: will holds informations about file/directory
//	- givenPath: selected directory's path
func SetNodeSelected(tree *tview.TreeView, mainTable *tview.Table, pages *tview.Pages, fileDirData cmap.ConcurrentMap, givenPath string) {

	tree.SetSelectedFunc(func(selectedNode *tview.TreeNode) {

		nodeReference := selectedNode.GetReference()

		// If the file/directory not exist anymore
		if _, err := os.Lstat(nodeReference.(FileDirStruct).FullPath); os.IsNotExist(err) {

			// Display error page
			notExistPage := createNotExistPage(nodeReference.(FileDirStruct).FullPath, pages, "mainPage")
			pages.RemovePage("notExistPage")
			pages.AddAndSwitchToPage("notExistPage", notExistPage, true)
			return
		}

		// Display informations about files and subdirectories under the selected directory
		UpdateTableChildren(mainTable, pages, fileDirData, nodeReference.(FileDirStruct).FullPath, givenPath)

		// Refresh children nodes of the selected directory (to be always updated)
		selectedNode.ClearChildren()
		AddNodes(selectedNode, nodeReference.(FileDirStruct).FullPath)
		selectedNode.SetExpanded(!selectedNode.IsExpanded())
	})
}

// Refresh table content to update files and directories list (at selection or after file/directory deletion)
//	- mainTable: table list containing selected folder's content
//	- pages: holds all pages for this application
//	- fileDirData: will holds informations about file/directory
//	- dirPath: parent directory's path of the selected file/directory
//	- givenPath: selected file/directory's path
func UpdateTableChildren(mainTable *tview.Table, pages *tview.Pages, fileDirData cmap.ConcurrentMap, dirPath string, givenPath string) {

	// If the parent directory doesn't exist, do nothing
	_, err := os.Lstat(dirPath)
	if err != nil {
		return
	}

	mainTable.Clear()

	directChildrenSlice, haveChild := getDirectChildrenDir(dirPath, fileDirData)

	if haveChild {
		for i := 0; i < len(directChildrenSlice); i++ {
			fileDirSet, _ := fileDirData.Get(directChildrenSlice[i].FullPath)
			fp := fileDirSet.(FileDirStruct).FullPath
			fileDirStat, _ := os.Lstat(fp)

			textColor := tcell.ColorWhite
			if fileDirSet.(FileDirStruct).IsDir {
				textColor = tcell.ColorGreen
			}

			mainTable.SetCell(i, 0, tview.NewTableCell(fileDirStat.Mode().String()).SetTextColor(textColor))
			mainTable.SetCell(i, 1, tview.NewTableCell(path.Base(fp)).SetTextColor(textColor))
			mainTable.SetCell(i, 2, tview.NewTableCell(humanize.Bytes(fileDirSet.(FileDirStruct).Size)).SetTextColor(textColor))

			// Display detail page about the selected file/directory from the table
			mainTable.SetSelectedFunc(func(row int, column int) {

				// If the file/directory doesn't exist anymore, create error page and do Return immediately
				_, err := os.Lstat(fp)
				if err != nil {
					notExistPage := createNotExistPage(fp, pages, "mainPage")
					pages.RemovePage("notExistPage")
					pages.AddAndSwitchToPage("notExistPage", notExistPage, true)
					return
				}

				// Create/Refresh file/directory properties page
				usPropPage := CreatePropPage(directChildrenSlice[row], "mainPage", pages, fileDirData, mainTable, givenPath)

				// No way to refresh, so delete and create
				pages.RemovePage("propertiesPage")
				pages.AddAndSwitchToPage("propertiesPage", usPropPage, true)
			})
		}
	}

}

// Return direct children files and directories list for the path given in parameter
//	- dirPath: directory's path to get children
//	- cDirFilesMap: will holds informations about file/directory
func getDirectChildrenDir(dirPath string, cDirFilesMap cmap.ConcurrentMap) ([]FileDirStruct, bool) {

	directChildrenSlice := make([]FileDirStruct, 1)
	childExist := false
	for _, k := range cDirFilesMap.Keys() {
		if path.Dir(k) == dirPath {
			fileDirSet, _ := cDirFilesMap.Get(k)

			// Initialize first slice's data
			if !childExist {
				directChildrenSlice[0], _ = fileDirSet.(FileDirStruct)
				childExist = true
			} else {
				directChildrenSlice = append(directChildrenSlice, fileDirSet.(FileDirStruct))
			}
		}
	}

	// Sort result by size
	sort.Slice(directChildrenSlice, func(i, j int) bool { return directChildrenSlice[i].Size > directChildrenSlice[j].Size })

	return directChildrenSlice, childExist
}

// Update header when user navigate into the tree
//	- tree: navigation tree
//	- headerInfo: header component to display full path of selected directory from the tree
func OnNodeChanged(tree *tview.TreeView, headerInfo *tview.Table) {
	tree.SetChangedFunc(func(focusedNode *tview.TreeNode) {
		nodeReference := focusedNode.GetReference()

		if nodeReference != nil {
			headerInfo.Clear()
			headerInfo.SetCell(0, 0, tview.NewTableCell(nodeReference.(FileDirStruct).FullPath).SetTextColor(tcell.ColorGreen))
		}
	})
}

// Display error page when a selected folder not exist anymore
//	- fullPath: full path of the missing file/directory
//	- pages: holds all pages for this application
//	- nextPage: holds reference of the next page
func createNotExistPage(fullPath string, pages *tview.Pages, nextPage string) *tview.Flex {
	notExistTable := tview.NewTable().SetSelectable(false, false)
	notExistTable.SetCell(0, 0, tview.NewTableCell("Error!").SetTextColor(tcell.ColorRed)).
		SetCell(2, 0, tview.NewTableCell(fullPath).SetTextColor(tcell.ColorGreen)).
		SetCell(3, 0, tview.NewTableCell("doesn't exist anymore!").SetTextColor(tcell.ColorRed))

	form := tview.NewForm().AddButton("OK", func() {
		pages.SwitchToPage(nextPage)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(notExistTable, 0, 1, false).AddItem(form, 0, 2, true)
	return flex
}
