package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDictionaryService_CRUD(t *testing.T) {
	// Setup
	db := SetupTestDB()
	service := NewDictionaryService(db)

	// 1. Create Dictionary
	keySameAsValue := false
	req := CreateDictionaryRequest{
		Code:           "device_status",
		Name:           "设备状态",
		Module:         "device",
		Description:    "设备的运行状态",
		KeySameAsValue: &keySameAsValue,
		SortOrder:      1,
	}
	dict, err := service.CreateDictionary(req)
	assert.NoError(t, err)
	assert.NotNil(t, dict)
	assert.Equal(t, "device_status", dict.Code)
	assert.Equal(t, "设备状态", dict.Name)
	assert.False(t, dict.KeySameAsValue)

	// 2. Create Dictionary Items
	itemReq1 := CreateDictionaryItemRequest{
		Key:       "1",
		Value:     "在线",
		IsDefault: true,
		SortOrder: 1,
	}
	item1, err := service.CreateDictionaryItem(dict.ID, itemReq1)
	assert.NoError(t, err)
	assert.Equal(t, "1", item1.Key)
	assert.Equal(t, "在线", item1.Value)

	itemReq2 := CreateDictionaryItemRequest{
		Key:       "2",
		Value:     "离线",
		SortOrder: 2,
	}
	item2, err := service.CreateDictionaryItem(dict.ID, itemReq2)
	assert.NoError(t, err)
	assert.Equal(t, "2", item2.Key)

	// 3. Get Dictionary with Items
	detail, err := service.GetDictionary(dict.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(detail.Items))
	assert.Equal(t, "在线", detail.Items[0].Value)

	// 4. Get Items by Code (Common API)
	items, err := service.GetItemsByCode("device_status")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(items))
	assert.Equal(t, "1", items[0].Key)
	assert.Equal(t, "在线", items[0].Value)

	// 5. Update Dictionary
	newName := "设备运行状态"
	updateReq := UpdateDictionaryRequest{
		Name: newName,
	}
	updatedDict, err := service.UpdateDictionary(dict.ID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, newName, updatedDict.Name)

	// 6. Update Dictionary Item
	newValue := "Online"
	updateItemReq := UpdateDictionaryItemRequest{
		Value: newValue,
	}
	updatedItem, err := service.UpdateDictionaryItem(item1.ID, updateItemReq)
	assert.NoError(t, err)
	assert.Equal(t, newValue, updatedItem.Value)

	// 7. Delete Item
	err = service.DeleteDictionaryItem(item2.ID)
	assert.NoError(t, err)

	itemsAfterDelete, err := service.GetItemsByCode("device_status")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(itemsAfterDelete))

	// 8. Delete Dictionary
	err = service.DeleteDictionary(dict.ID)
	assert.NoError(t, err)

	// Verify cascade delete (logic is in DB constraint, but mocks might behavior differently depending on driver)
	// For this test, we just check service returns nil/empty
	nonExistent, err := service.GetDictionary(dict.ID)
	assert.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestDictionaryService_BatchUpdate(t *testing.T) {
	db := SetupTestDB()
	service := NewDictionaryService(db)

	// Create Dictionary
	req := CreateDictionaryRequest{
		Code: "batch_test",
		Name: "Batch Test",
	}
	dict, _ := service.CreateDictionary(req)

	// Initial Items
	service.CreateDictionaryItem(dict.ID, CreateDictionaryItemRequest{Key: "A", Value: "A"})
	itemB, _ := service.CreateDictionaryItem(dict.ID, CreateDictionaryItemRequest{Key: "B", Value: "B"})

	// Batch Update
	// 1. Update A -> A_Modified
	// 2. Delete B
	// 3. Create C
	idA := uint64(1) // Assuming ID sequence, but better to fetch
	items, _ := service.GetItemsByCode("batch_test")
	idA = items[0].ID
	idB := itemB.ID // pointer safe

	batchReq := BatchUpdateDictionaryItemsRequest{
		Items: []DictionaryItemBatchItem{
			{ID: &idA, Key: "A", Value: "A_Modified", IsEnabled: true},
			{ID: &idB, Key: "B", Value: "B", Delete: true},
			{Key: "C", Value: "C", SortOrder: 3, IsEnabled: true},
		},
	}

	err := service.BatchUpdateItems(dict.ID, batchReq)
	assert.NoError(t, err)

	// Verify
	finalItems, err := service.GetItemsByCode("batch_test")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(finalItems))

	// Map generic check
	itemMap := make(map[string]string)
	for _, item := range finalItems {
		itemMap[item.Key] = item.Value
	}

	assert.Equal(t, "A_Modified", itemMap["A"])
	assert.Equal(t, "C", itemMap["C"])
	assert.NotContains(t, itemMap, "B")
}

func TestDictionaryService_DuplicateCode(t *testing.T) {
	db := SetupTestDB()
	service := NewDictionaryService(db)

	req := CreateDictionaryRequest{Code: "dup", Name: "Duplicate"}
	_, err := service.CreateDictionary(req)
	assert.NoError(t, err)

	_, err = service.CreateDictionary(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
}

func TestDictionaryService_DuplicateItemKey(t *testing.T) {
	db := SetupTestDB()
	service := NewDictionaryService(db)

	dict, _ := service.CreateDictionary(CreateDictionaryRequest{Code: "test", Name: "Test"})

	_, err := service.CreateDictionaryItem(dict.ID, CreateDictionaryItemRequest{Key: "K1", Value: "V1"})
	assert.NoError(t, err)

	_, err = service.CreateDictionaryItem(dict.ID, CreateDictionaryItemRequest{Key: "K1", Value: "V2"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已存在")
}

// 模拟 SetupTestDB，如果项目中已有 helper 则不需要复制
// 这里假设 sharedservices 包下已有 SetupTestDB，直接引用
