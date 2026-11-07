package world

import (
	_ "embed"
	"fmt"
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"image"
)

// Item represents an item that may be added to an inventory. It has a method to encode the item to an ID and
// a metadata value.
type Item interface {
	// EncodeItem encodes the item to its Minecraft representation, which consists of a numerical ID and a
	// metadata value.
	EncodeItem() (name string, meta int16)
}

// CustomItem represents an item that is non-vanilla and requires a resource pack and extra steps to show it
// to the client.
type CustomItem interface {
	Item
	// Name is the name that will be displayed on the item to all clients.
	Name() string
	// Texture is the Image of the texture for this item.
	Texture() image.Image
	// Category is the category the item will be listed under in the creative inventory.
	Category() category.Category
}

// RegisterItem registers an item with the ID and meta passed. Once registered, items may be obtained from an
// ID and metadata value using itemByID().
// If an item with the ID and meta passed already exists, RegisterItem panics.
func RegisterItem(item Item) {
	name, meta := item.EncodeItem()
	h := itemHash{name: name, meta: meta}

	if _, ok := items[h]; ok {
		panic(fmt.Sprintf("item registered with name %v and meta %v already exists", name, meta))
	}
	if c, ok := item.(CustomItem); ok {
		nextRID := int32(len(itemNamesToRuntimeIDs))
		itemRuntimeIDsToNames[nextRID] = name
		itemNamesToRuntimeIDs[name] = nextRID

		customItems = append(customItems, c)
	}
	if _, ok := itemNamesToRuntimeIDs[name]; !ok {
		panic(fmt.Sprintf("item name %v does not have a runtime ID", name))
	}
	items[h] = item
}

// itemHash is a combination of an item's name and metadata. It is used as a key in hash maps.
type itemHash struct {
	name string
	meta int16
}

var (
	//go:embed vanilla_items.nbt
	itemRuntimeIDData []byte
	// items holds a list of all registered items, indexed using the itemHash created when calling
	// Item.EncodeItem.
	items = map[itemHash]Item{}
	// customItems holds a list of all registered custom items.
	customItems []CustomItem
	// itemRuntimeIDsToNames holds a map to translate item runtime IDs to string IDs.
	itemRuntimeIDsToNames = map[int32]string{}
	// itemNamesToRuntimeIDs holds a map to translate item string IDs to runtime IDs.
	itemNamesToRuntimeIDs = map[string]int32{}
)

// init reads all item entries from the resource JSON, and sets the according values in the runtime ID maps.
func init() {
	var m map[string]struct {
		RuntimeID      int32          `nbt:"runtime_id"`
		ComponentBased bool           `nbt:"component_based"`
		Version        int32          `nbt:"version"`
		Data           map[string]any `nbt:"data,omitempty"`
	}
	err := nbt.Unmarshal(itemRuntimeIDData, &m)
	if err != nil {
		panic(err)
	}
	for name, e := range m {
		itemNamesToRuntimeIDs[name] = e.RuntimeID
		itemRuntimeIDsToNames[e.RuntimeID] = name
	}
}

// ItemByName attempts to return an item by a name and a metadata value. If the
// item is a known vanilla item but has no dedicated Dragonfly implementation,
// an inert vanillaItemFallback is returned instead. This allows vanilla items
// to remain visible in creative inventories and retain their identity in
// inventories, commands and network packets while their behaviour is not yet
// implemented.
func ItemByName(name string, meta int16) (Item, bool) {
	it, ok := items[itemHash{name: name, meta: meta}]
	if !ok {
		// Also try obtaining the item with a metadata value of 0, for cases with durability.
		it, ok = items[itemHash{name: name}]
	}
	if ok {
		return it, true
	}
	if _, vanilla := itemNamesToRuntimeIDs[name]; vanilla {
		return vanillaItemFallback{name: name, meta: meta}, true
	}
	return nil, false
}

// vanillaItemFallback preserves the identity of a vanilla item that has a
// runtime ID but no dedicated item implementation. It intentionally does not
// implement any use, equipment, durability or interaction interfaces.
type vanillaItemFallback struct {
	name string
	meta int16
}

func (i vanillaItemFallback) EncodeItem() (string, int16) {
	return i.name, i.meta
}

// ItemRuntimeID attempts to return the runtime ID of the Item passed. False is returned if the Item is not
// registered.
func ItemRuntimeID(i Item) (rid int32, meta int16, ok bool) {
	name, meta := i.EncodeItem()
	rid, ok = itemNamesToRuntimeIDs[name]
	return rid, meta, ok
}

// ItemByRuntimeID attempts to return an Item by the runtime ID passed. If no item with that runtime ID exists,
// false is returned. ItemByRuntimeID also tries to find the item with a metadata value of 0.
func ItemByRuntimeID(rid int32, meta int16) (Item, bool) {
	name, ok := itemRuntimeIDsToNames[rid]
	if !ok {
		return nil, false
	}
	return ItemByName(name, meta)
}

// Items returns a slice of all registered items.
func Items() []Item {
	m := make([]Item, 0, len(items))
	for _, i := range items {
		m = append(m, i)
	}
	return m
}

// CustomItems returns a slice of all registered custom items.
func CustomItems() []CustomItem {
	return customItems
}
