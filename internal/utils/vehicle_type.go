package utils

import "strings"

// MapVehicleTypeIDForSpace maps vehicle type IDs for space logic.
// Both car and suv should use the same space pool (car's), motorcycle stays separate.
func MapVehicleTypeIDForSpace(vehicleTypeID int, vehicleTypeName string) int {
	// Example: car = 1, suv = 3, motorcycle = 2
	// You may want to dynamically fetch this mapping from DB if IDs are not stable
	if strings.ToLower(vehicleTypeName) == "suv" {
		return 1 // car's ID
	}
	return vehicleTypeID
}

// VehicleTypeIDsForSpace returns all vehicle_type_ids that share the same space pool as the given type name.
func VehicleTypeIDsForSpace(vehicleTypes []struct{ID int; Name string}, vehicleTypeName string) []int {
	var ids []int
	name := strings.ToLower(vehicleTypeName)
	for _, vt := range vehicleTypes {
		if name == "car" || name == "suv" {
			if vt.Name == "car" || vt.Name == "suv" {
				ids = append(ids, vt.ID)
			}
		} else if vt.Name == name {
			ids = append(ids, vt.ID)
		}
	}
	return ids
}
