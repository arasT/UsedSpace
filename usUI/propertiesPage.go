// Create Dialog box like:
//	- properties page
//	- delete page
//	- error page
package usUI

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/djherbis/times"
	"github.com/dustin/go-humanize"
	"github.com/gdamore/tcell"
	"github.com/orcaman/concurrent-map"
	"github.com/rivo/tview"
)

// Create properties page for selected files/directories
//	- fileDir: holds data of the file/directory to get properties
//	- nextPage: reference of the next page
//	- pages: holds all pages for this application
//	- fileDirData: will holds informations about file/directory
//	- mainTable: table list containing selected folder's content
//	- givenPath: directory's path to scan
func CreatePropPage(fileDir FileDirStruct, nextPage string, pages *tview.Pages, fileDirData cmap.ConcurrentMap, mainTable *tview.Table, givenPath string) *tview.Flex {
	propTable := tview.NewTable().SetSelectable(false, false)

	fdInfo := getFileDirInfo(fileDir)
	propTable.
		//SetCell(0, 0, tview.NewTableCell(" Full Path").SetTextColor(tcell.ColorGreen)).SetCellSimple(0, 1, ": "+fdInfo["fullPath"]).
		SetCell(0, 0, tview.NewTableCell(" Name").SetTextColor(tcell.ColorGreen)).SetCellSimple(0, 1, ": "+fdInfo["name"]).
		SetCell(1, 0, tview.NewTableCell(" Type").SetTextColor(tcell.ColorGreen)).SetCellSimple(1, 1, ": "+fdInfo["type"]).
		SetCell(2, 0, tview.NewTableCell(" Parent Folder").SetTextColor(tcell.ColorGreen)).SetCellSimple(2, 1, ": "+fdInfo["parent"]).
		SetCell(3, 0, tview.NewTableCell(" Size").SetTextColor(tcell.ColorGreen)).SetCellSimple(3, 1, ": "+fdInfo["size"]).
		SetCell(4, 0, tview.NewTableCell(" Last Access").SetTextColor(tcell.ColorGreen)).SetCellSimple(4, 1, ": "+fdInfo["accessTime"]).
		SetCell(5, 0, tview.NewTableCell(" Last Modification").SetTextColor(tcell.ColorGreen)).SetCellSimple(5, 1, ": "+fdInfo["modTime"])

	// Add Contents row for directory object
	if fileDir.IsDir {
		propTable.SetCell(5, 0, tview.NewTableCell(" Contents").SetTextColor(tcell.ColorGreen)).SetCellSimple(5, 1, ": "+fdInfo["content"])
	}

	form := tview.NewForm().
		AddButton("OK", func() {
			pages.SwitchToPage(nextPage)
		}).
		AddButton("Delete", func() {

			// Create confirm delete page
			delPage := createDelPage(fileDir, "propertiesPage", pages, fileDirData, mainTable, givenPath)

			// No way to refresh, so delete and create
			pages.RemovePage("confirmDelPage")
			pages.AddAndSwitchToPage("confirmDelPage", delPage, true)
		})

	propTitle := tview.NewTextView().SetScrollable(false).SetText("Properties").SetTextColor(tcell.ColorBlue)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(propTitle, 2, 1, false).AddItem(propTable, 0, 1, false).AddItem(form, 0, 2, true)

	return flex
}

// Create delete page confirmation, delete the file/directory and update stored data
//	- fileDir: holds data of the file/directory to get properties
//	- nextPage: reference of the next page
//	- pages: holds all pages for this application
//	- fileDirData: will holds informations about file/directory
//	- mainTable: table list containing selected folder's content (to be updated)
//	- givenPath: path of the scanned directory
func createDelPage(fileDir FileDirStruct, nextPage string, pages *tview.Pages, fileDirData cmap.ConcurrentMap, mainTable *tview.Table, givenPath string) *tview.Flex {
	delTable := tview.NewTable().SetSelectable(false, false)
	delTable.SetCell(0, 0, tview.NewTableCell("Are you sure to delete: ").SetTextColor(tcell.ColorRed)).
		SetCell(2, 0, tview.NewTableCell(fileDir.FullPath))

	form := tview.NewForm().AddButton("OK", func() {

		// Delete the file/directory
		err := os.Remove(fileDir.FullPath)
		if fileDir.IsDir {
			err = os.RemoveAll(fileDir.FullPath)
		}

		if err != nil {
			//panic(err)
			errorPage := createErrorPage(fileDir, err.Error(), pages, "propertiesPage")
			pages.RemovePage("errorPage")
			pages.AddAndSwitchToPage("errorPage", errorPage, true)
		} else {

			// Update all directories Size
			currentParent := fileDir.FullPath
			for currentParent != givenPath {
				currentParent = path.Dir(currentParent)

				currentParentObj, _ := fileDirData.Get(currentParent)
				currentParentSize := currentParentObj.(FileDirStruct).Size
				fileDirData.Set(currentParent, FileDirStruct{currentParent, currentParentSize - fileDir.Size, true})
			}
			fileDirData.Remove(fileDir.FullPath) // Remove its instance from memory

			// Refresh file/directory table for the parent directory into main page then switch to it
			UpdateTableChildren(mainTable, pages, fileDirData, path.Dir(fileDir.FullPath), givenPath)
			pages.SwitchToPage("mainPage")
		}
	}).
		AddButton("Cancel", func() {
			pages.SwitchToPage(nextPage)
		})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(delTable, 0, 1, false).AddItem(form, 0, 2, true)
	return flex
}

// Create error page if a file/directory cannot be removed
//	- fileDir: holds data of the file/directory to get properties
//	- errorMsg: error message
//	- pages: holds all pages for this application
//	- nextPage: reference of the next page
func createErrorPage(fileDir FileDirStruct, errorMsg string, pages *tview.Pages, nextPage string) *tview.Flex {
	reason := strings.Split(errorMsg, ":")
	if len(reason[1]) == 0 {
		reason[1] = "Unknown Reason"
	}
	errorTable := tview.NewTable().SetSelectable(false, false)
	errorTable.SetCell(0, 0, tview.NewTableCell("Error!").SetTextColor(tcell.ColorRed)).
		SetCell(2, 0, tview.NewTableCell(fileDir.FullPath)).
		SetCell(3, 0, tview.NewTableCell("can't be removed : "+reason[1]).SetTextColor(tcell.ColorRed))

	form := tview.NewForm().AddButton("OK", func() {
		pages.SwitchToPage(nextPage)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(errorTable, 0, 1, false).AddItem(form, 0, 2, true)
	return flex
}

// Return more informations about a selected file/directory
//	- fileDir: holds data of the file/directory to get properties
func getFileDirInfo(fileDir FileDirStruct) map[string]string {
	var fileDirInfo = make(map[string]string)

	// Get time informations about the file/directory
	fdInfoTime, _ := times.Stat(fileDir.FullPath)
	accesTimeStr := fdInfoTime.AccessTime().String()
	accesTimeDatePart := strings.Split(accesTimeStr, ".")[0]
	accesTimeGMTPart := strings.Split(accesTimeStr, " ")[2] + " " + strings.Split(accesTimeStr, " ")[3]
	modTimeStr := fdInfoTime.ModTime().String()
	modTimeDatePart := strings.Split(modTimeStr, ".")[0]
	modTimeGMTPart := strings.Split(modTimeStr, " ")[2] + " " + strings.Split(modTimeStr, " ")[3]

	// Get type of the file/directory
	fi, _ := os.Lstat(fileDir.FullPath)
	fileDirInfo["type"] = "Unknown type"

	switch mode := fi.Mode(); {
	case mode.IsRegular():
		fileDirInfo["type"] = "File"
	case mode.IsDir():
		fileDirInfo["type"] = "Directory"
	case mode&os.ModeSymlink != 0:
		fileDirInfo["type"] = "Symbolic link"
	case mode&os.ModeNamedPipe != 0:
		fileDirInfo["type"] = "Named pipe"
	}

	//fileDirInfo["fullPath"] = fileDir.FullPath
	fileDirInfo["name"] = path.Base(fileDir.FullPath)
	fileDirInfo["size"] = humanize.Bytes(fileDir.Size)
	fileDirInfo["parent"] = path.Dir(fileDir.FullPath)
	fileDirInfo["accessTime"] = humanize.Time(fdInfoTime.AccessTime()) + " (" + accesTimeDatePart + " " + accesTimeGMTPart + ")"
	fileDirInfo["modTime"] = humanize.Time(fdInfoTime.ModTime()) + " (" + modTimeDatePart + " " + modTimeGMTPart + ")"

	// If it is a directory, count children and add content fields
	if fileDirInfo["type"] == "Directory" {
		file, _ := os.Open(fileDir.FullPath)
		defer file.Close()
		childrenList, _ := file.Readdirnames(0)

		childrenDesc := " element"
		if len(childrenList) >= 2 {
			childrenDesc = childrenDesc + "s"
		}
		fileDirInfo["content"] = strconv.Itoa(len(childrenList)) + childrenDesc
	}

	return fileDirInfo
}
