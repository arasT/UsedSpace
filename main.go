package main

import (
	"os"
	"path"

	"github.com/gdamore/tcell"
	"github.com/orcaman/concurrent-map"
	"github.com/rivo/tview"

	usUI "UsedSpace/usUI"
	usWalk "UsedSpace/usWalk"
)

// Create the application instance, all main components, start the app and directory scan in parallel
func main() {
	givenPath, _ := os.Getwd() // Without arguments, scan the current directory
	if len(os.Args) > 2 {
		panic("Too much arguments!!")
	} else {
		if len(os.Args) == 2 {
			givenPath = os.Args[1]
			latestChar := givenPath[len(givenPath)-1:]

			// There is a problem yet while scanning root path. TO BE RESOLVED
			if givenPath == "/" {
				panic("Root path cannot be scanned yet for this version.")
			}

			// If the user give a path With "/" at end, remove it
			// Root ("/" only) not considered
			if len(givenPath) > 1 && latestChar == string(os.PathSeparator) {
				givenPath = path.Dir(givenPath)
			}

			// Exit if the user given a file instead of a directory path
			if fd, _ := os.Lstat(givenPath); !fd.IsDir() {
				panic("You have to give a directory path!!")
			}

			// Exit if the directory path doesn't exist
			if _, err := os.Lstat(givenPath); os.IsNotExist(err) {
				panic(err)
			}
		}
	}

	// Init variable holding informations about scanned files and directories
	cDirFilesMap := cmap.New()

	// Init the main app
	usApp := tview.NewApplication()

	// Create page holding all pages
	usPages := tview.NewPages()

	// Create header for the main layout
	//usHeader := tview.NewTextView().SetScrollable(false).SetText(givenPath)
	usHeader := tview.NewTable().SetSelectable(false, false)
	usHeader.SetCell(0, 0, tview.NewTableCell(givenPath).SetTextColor(tcell.ColorGreen))

	// Create footer for the main layout
	usFooter := tview.NewTextView().SetScrollable(false).SetText("(!) Directions to navigate / TAB to switch between buttons / CTRL+C to quit").
		SetTextColor(tcell.ColorBlue)

	// Create Waiting page (displayed until scan finished)
	usWaitingTable := tview.NewTable().SetSelectable(false, false)

	// Create navigation tree, and intialize its root node
	rootNode := tview.NewTreeNode(path.Base(givenPath)).SetColor(tcell.ColorGreen).
		SetReference(usUI.FileDirStruct{givenPath, uint64(0), true})
	usTree := tview.NewTreeView().SetRoot(rootNode).SetCurrentNode(rootNode)

	// Create table displaying files and directories into the selected folder from the tree
	usTable := tview.NewTable()
	usTable.SetSelectable(true, false)

	// Add the directory's path to scan to the root node
	usUI.AddNodes(rootNode, givenPath)

	// Update Header each time user navigate into the tree view
	usUI.OnNodeChanged(usTree, usHeader)

	// If a directory was selected, open it.
	usUI.SetNodeSelected(usTree, usTable, usPages, cDirFilesMap, givenPath)

	// Set up the container for main page
	usMainPage := usUI.SetUpMainPage(usTree, usTable)

	// Display waiting page ...
	usWaitingTable.SetCellSimple(0, 0, "Scanning "+givenPath).SetCellSimple(2, 0, "Please wait ...")
	usPages.AddAndSwitchToPage("waitingPage", usWaitingTable, true)

	// Start scan in parallel then display main page immediately when scan is done
	go waitingScan(usApp, usPages, usMainPage, usTable, usTree, usWaitingTable, givenPath, cDirFilesMap)

	// General keys binding
	usApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		// Switch between tab (usTree, usTable) when user press Right and Left keys
		if usMainPage.HasFocus() {
			if event.Key() == tcell.KeyRight {
				usApp.SetFocus(usTable)
				return nil // Don't propagate right and left event handler to primitives into the main page
			}
			if event.Key() == tcell.KeyLeft {
				usApp.SetFocus(usTree)
				return nil // Don't propagate right and left event handler to primitives into the main page
			}
		} else { // Don't propagate Up and Down event handler to primitives for other pages
			if event.Key() == tcell.KeyUp {
				return nil
			}
			if event.Key() == tcell.KeyDown {
				return nil
			}
		}

		return event
	})

	// Create the main layout
	usLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(usHeader, 2, 1, false).
		AddItem(usPages, 0, 1, true).
		AddItem(usFooter, 1, 1, false)

	// Start the app
	if err := usApp.SetRoot(usLayout, true).Run(); err != nil {
		panic(err)
	}
}

// Scan given folder (path) in parallel then display the main page while scan is done
//	- usApp: the main application
//	- usPages: holds all pages for this application
//	- usMainPage: this application's main page component
//	- usTable: table list containing selected folder's content
//	- usTree: navigation tree
//	- usWaintingTable: Table displayed to wait until scan will be done
//	- givenPath: directory's path to scan
//	- cDirFilesMap: will holds informations about file/directory
func waitingScan(usApp *tview.Application, usPages *tview.Pages, usMainPage *tview.Flex, usTable *tview.Table, usTree *tview.TreeView, usWaintingTable *tview.Table, givenPath string, cDirFilesMap cmap.ConcurrentMap) {

	scanState := make(chan bool) // Channel to check if the scan is done
	go usWalk.WalkGivenDir(givenPath, cDirFilesMap, scanState)
	<-scanState

	// Display immediately table information about files/folders children
	usUI.UpdateTableChildren(usTable, usPages, cDirFilesMap, givenPath, givenPath)

	usPages.RemovePage("waitingPage")
	usPages.AddAndSwitchToPage("mainPage", usMainPage, true)
	usApp.Draw()           // Mandatory to refresh the current page, else we have to type some keys when scan finished
	usApp.SetFocus(usTree) // Set the focus to the tree
}
