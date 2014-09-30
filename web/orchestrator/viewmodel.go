// Copyright 2014 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package orchestrator

import (
	"fmt"
	"github.com/andreaskoch/allmark2/common/route"
	"github.com/andreaskoch/allmark2/model"
	"github.com/andreaskoch/allmark2/web/view/viewmodel"
)

type ViewModelOrchestrator struct {
	*Orchestrator

	navigationOrchestrator NavigationOrchestrator
	tagOrchestrator        TagsOrchestrator
	fileOrchestrator       FileOrchestrator
	locationOrchestrator   LocationOrchestrator
}

func (orchestrator *ViewModelOrchestrator) GetViewModel(itemRoute route.Route) (viewModel viewmodel.Model, found bool) {

	// get the requested item
	item := orchestrator.getItem(itemRoute)
	if item == nil {
		return viewModel, false
	}

	return orchestrator.getViewModel(item), true

}

func (orchestrator *ViewModelOrchestrator) GetLatest(itemRoute route.Route, pageSize, page int) (models []*viewmodel.Model, found bool) {

	leafes := orchestrator.getAllLeafes(itemRoute)
	viewmodel.SortModelBy(sortModelsByDate).Sort(leafes)

	return leafes, true
}

func (orchestrator *ViewModelOrchestrator) getViewModel(item *model.Item) viewmodel.Model {

	itemRoute := item.Route()

	// get the root item
	root := orchestrator.rootItem()
	if root == nil {
		panic(fmt.Sprintf("Cannot get viewmodel for route %q because no root item was found.", itemRoute))
	}

	// convert content
	convertedContent, err := orchestrator.converter.Convert(orchestrator.getItemByAlias, orchestrator.relativePather(itemRoute), item)
	if err != nil {
		orchestrator.logger.Warn("Cannot convert content for item %q. Error: %s.", item.String(), err.Error())
		convertedContent = "<!-- Conversion Error -->"
	}

	// create a view model
	viewModel := viewmodel.Model{
		Base:    getBaseModel(root, item, orchestrator.itemPather()),
		Content: convertedContent,
		Childs:  orchestrator.getChildModels(itemRoute),

		// navigation
		ToplevelNavigation:   orchestrator.navigationOrchestrator.GetToplevelNavigation(),
		BreadcrumbNavigation: orchestrator.navigationOrchestrator.GetBreadcrumbNavigation(itemRoute),

		// tags
		Tags: orchestrator.tagOrchestrator.getItemTags(itemRoute),

		// files
		Files: orchestrator.fileOrchestrator.GetFiles(itemRoute),

		// Locations
		Locations: orchestrator.locationOrchestrator.GetLocations(item.MetaData.Locations, func(i *model.Item) viewmodel.Model {
			return orchestrator.getViewModel(i)
		}),

		// Geo Coordinates
		GeoLocation: getGeoLocation(item),
	}

	// special viewmodel attributes
	isRepositoryItem := item.Type == model.TypeRepository
	if isRepositoryItem {

		// tag cloud
		repositoryIsNotEmpty := orchestrator.repository.Size() > 5 // don't bother to create a tag cloud if there aren't enough documents
		if repositoryIsNotEmpty {

			tagCloud := orchestrator.tagOrchestrator.GetTagCloud()
			viewModel.TagCloud = tagCloud

		}

	}

	return viewModel
}

func (orchestrator *ViewModelOrchestrator) getAllLeafes(parentRoute route.Route) []*viewmodel.Model {

	childModels := make([]*viewmodel.Model, 0)

	childItems := orchestrator.getChilds(parentRoute)
	if hasNoMoreChilds := len(childItems) == 0; hasNoMoreChilds {

		viewModel, found := orchestrator.GetViewModel(parentRoute)
		if !found {
			return []*viewmodel.Model{}
		}

		return []*viewmodel.Model{&viewModel}
	}

	// recurse
	for _, childItem := range childItems {
		childModels = append(childModels, orchestrator.getAllLeafes(childItem.Route())...)
	}

	return childModels

}

func (orchestrator *ViewModelOrchestrator) getChildModels(itemRoute route.Route) []*viewmodel.Base {

	rootItem := orchestrator.rootItem()
	if rootItem == nil {
		orchestrator.logger.Fatal("No root item found")
	}

	pathProvider := orchestrator.relativePather(itemRoute)

	childModels := make([]*viewmodel.Base, 0)

	childItems := orchestrator.getChilds(itemRoute)
	for _, childItem := range childItems {
		baseModel := getBaseModel(rootItem, childItem, pathProvider)
		childModels = append(childModels, &baseModel)
	}

	// sort the models
	viewmodel.SortBaseModelBy(sortBaseModelsByDate).Sort(childModels)

	return childModels
}

// sort the models by date and name
func sortBaseModelsByDate(model1, model2 *viewmodel.Base) bool {

	return model1.CreationDate > model2.CreationDate

}

// sort the models by date and name
func sortModelsByDate(model1, model2 *viewmodel.Model) bool {

	return model1.CreationDate > model2.CreationDate

}
