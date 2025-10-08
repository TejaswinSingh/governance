package governance

import (
	"context"
	"errors"
	"strings"

	"github.com/Resolyte-official/service-governance/pkg/auth"
	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"
	"github.com/Resolyte-official/service-governance/pkg/persist"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ServiceStore struct {
	Redis  *persist.RedisService
	Mongo  *persist.MongoService
	Casbin *auth.CasbinService
}

func uuidToBinary(id uuid.UUID) bson.Binary {
	return bson.Binary{Subtype: 0x04, Data: id[:]}
}

func uuidBinToString(id any) (string, bool) {
	bin, ok := id.(bson.Binary)
	if !ok || bin.Subtype != 0x04 || len(bin.Data) != 16 {
		return "", false
	}
	u, err := uuid.FromBytes(bin.Data)
	if err != nil {
		return "", false
	}
	return u.String(), true
}

func bsonDToMap(d bson.D) map[string]any {
	m := make(map[string]any)
	for _, elem := range d {
		m[elem.Key] = elem.Value
	}
	return m
}

func UUIDv5FromPhone(phone string, namespace uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(namespace, []byte(phone))
}

func (s *governanceService) getUser(ctx context.Context, userID uuid.UUID) (bson.M, error) {
	db := "metadata"
	coll := s.svcStore.Mongo.GetCollection(db, "users")

	// get user
	userDoc, err := s.getDoc(ctx, coll, bson.M{"_id": uuidToBinary(userID)})
	if err != nil {
		return nil, err
	}
	// check account status
	if status, ok := userDoc["account_status"].(string); !ok || !strings.EqualFold(status, "active") {
		return nil, myerrors.ErrAccessDenied.WithInternalErr(errors.New("user account status is not set to active"))
	}
	return userDoc, nil
}

func (s *governanceService) getDoc(ctx context.Context, coll *mongo.Collection, filter any) (bson.M, error) {
	res, err := s.svcStore.Mongo.FindOne(ctx, coll, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, myerrors.ErrMissingMetaData.WithInternalErr(err)
		}
		return nil, err
	}
	var doc bson.M
	if err := res.Decode(&doc); err != nil {
		return nil, myerrors.ErrMongo.WithInternalErr(err)
	}
	return doc, nil
}

func (s *governanceService) fetchUserMetaData(ctx context.Context, userID uuid.UUID) (bson.M, *auth.OtherClaims, error) {
	db := "metadata"
	collUsers := s.svcStore.Mongo.GetCollection(db, "users")

	// Fetch user
	userDoc, err := s.getDoc(ctx, collUsers, bson.M{"_id": uuidToBinary(userID)})
	if err != nil {
		return nil, nil, err
	}

	// check role exists
	role, ok := userDoc["role"].(string)
	if !ok || role == "" {
		return nil, nil, myerrors.ErrMissingMetaData
	}

	// check account status
	if status, ok := userDoc["account_status"].(string); !ok || !strings.EqualFold(status, "active") {
		return nil, nil, myerrors.ErrAccessDenied.WithInternalErr(errors.New("user account status is not set to active"))
	}

	if strings.EqualFold(role, string(auth.RoleParent)) {
		doc, ot, err := s.fetchParentMetaData(ctx, userDoc)
		if err != nil {
			return nil, nil, err
		}
		return doc, ot, nil

	} else if strings.EqualFold(role, string(auth.RoleGuardian)) {
		doc, ot, err := s.fetchGuardianMetaData(ctx, userDoc)
		if err != nil {
			return nil, nil, err
		}
		return doc, ot, nil
	}
	return nil, nil, myerrors.ErrMetaData
}

func (s *governanceService) fetchParentMetaData(ctx context.Context, userDoc bson.M) (bson.M, *auth.OtherClaims, error) {
	db := "metadata"
	collStudents := s.svcStore.Mongo.GetCollection(db, "students")
	collVehicles := s.svcStore.Mongo.GetCollection(db, "vehicles")
	collRoutes := s.svcStore.Mongo.GetCollection(db, "routes")

	val := struct {
		LinkedStudents    []string `json:"linked_students"`
		DelegatedStudents []string `json:"delegated_students"`
	}{
		LinkedStudents:    []string{},
		DelegatedStudents: []string{},
	}
	ot := &auth.OtherClaims{
		Name: "parent_claims",
	}

	// Fetch all students linked with the user
	studIDs, ok := userDoc["linked_ids"].(bson.A)
	if !ok || len(studIDs) <= 0 {
		return nil, nil, myerrors.ErrMetaData
	}
	var students []bson.M
	for _, l := range studIDs {

		// Fetch student
		studUUID, ok := l.(bson.Binary)
		if !ok || studUUID.Subtype != 0x04 {
			return nil, nil, myerrors.ErrMetaData
		}
		studDoc, err := s.getDoc(ctx, collStudents, bson.M{"_id": studUUID})
		if err != nil {
			return nil, nil, err
		}

		// convert binary UUIDs to string
		if uuidStr, ok := uuidBinToString(studDoc["_id"]); ok {
			studDoc["_id"] = uuidStr
			val.LinkedStudents = append(val.LinkedStudents, uuidStr)
		}
		if uuidStr, ok := uuidBinToString(studDoc["parent_id"]); ok {
			studDoc["parent_id"] = uuidStr
		}

		// Check if student has opted for surveillance
		hasSurveillance, ok := studDoc["has_surveillance"].(bool)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		if !hasSurveillance {
			studDoc["surveillance"] = nil
			students = append(students, studDoc)
			continue
		}
		d, ok := studDoc["surveillance"].(bson.D)
		if !ok || d == nil || len(d) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		surveillance := bsonDToMap(d)

		// Fetch vehicle assigned to student
		vehID, ok := surveillance["vehicle_id"].(bson.ObjectID)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		vehicleDoc, err := s.getDoc(ctx, collVehicles, bson.M{"_id": vehID})
		if err != nil {
			return nil, nil, err
		}

		// Fetch routes assigned to student
		assignedRoutes, ok := surveillance["assigned_route_ids"].(bson.A)
		if !ok || len(assignedRoutes) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		var routeDocs []bson.M
		for _, r := range assignedRoutes {
			routeID, ok := r.(bson.ObjectID)
			if !ok {
				return nil, nil, myerrors.ErrMetaData
			}
			routeDoc, err := s.getDoc(ctx, collRoutes, bson.M{"_id": routeID})
			if err != nil {
				return nil, nil, err
			}
			routeDocs = append(routeDocs, routeDoc)
		}

		// replace ids with embedded documents
		surveillance["assigned_route_ids"] = routeDocs
		surveillance["vehicle_id"] = vehicleDoc
		studDoc["surveillance"] = surveillance
		students = append(students, studDoc)
	}
	userDoc["linked_ids"] = students

	// convert binary UUID to string
	if uuidStr, ok := uuidBinToString(userDoc["_id"]); ok {
		userDoc["_id"] = uuidStr
	}
	if acc, ok := userDoc["sub_account"].(bson.D); ok && len(acc) == 2 {
		// acc[0] -> user_id
		// acc[1] -> delegated_ids
		if uuidStr, ok := uuidBinToString(acc[0].Value); ok {
			acc[0].Value = uuidStr
		}
		if arr, ok := acc[1].Value.(bson.A); ok {
			for i, id := range arr {
				if uuidStr, ok := uuidBinToString(id); ok {
					arr[i] = uuidStr
				}
			}
			acc[1].Value = arr
		}
		userDoc["sub_account"] = acc
	}

	superAcc, ok := userDoc["super_account"].(bson.D)
	if !ok || len(superAcc) != 2 {
		ot.Val = val
		return userDoc, ot, nil
	}
	// superAcc[0] -> user_id
	if uuidStr, ok := uuidBinToString(superAcc[0].Value); ok {
		superAcc[0].Value = uuidStr
	}
	// superAcc[1] -> delegated_ids
	delegatedIDs, ok := superAcc[1].Value.(bson.A)
	if !ok || len(delegatedIDs) <= 0 {
		return nil, nil, myerrors.ErrMetaData
	}
	students = nil
	// Fetch all students delegated to the user
	for _, l := range delegatedIDs {

		// Fetch student
		studUUID, ok := l.(bson.Binary)
		if !ok || studUUID.Subtype != 0x04 {
			return nil, nil, myerrors.ErrMetaData
		}
		studDoc, err := s.getDoc(ctx, collStudents, bson.M{"_id": studUUID})
		if err != nil {
			return nil, nil, err
		}

		// convert binary UUIDs to string
		if uuidStr, ok := uuidBinToString(studDoc["_id"]); ok {
			studDoc["_id"] = uuidStr
			val.DelegatedStudents = append(val.DelegatedStudents, uuidStr)
		}
		if uuidStr, ok := uuidBinToString(studDoc["parent_id"]); ok {
			studDoc["parent_id"] = uuidStr
		}

		// Check if student has opted for surveillance
		hasSurveillance, ok := studDoc["has_surveillance"].(bool)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		if !hasSurveillance {
			studDoc["surveillance"] = nil
			students = append(students, studDoc)
			continue
		}
		d, ok := studDoc["surveillance"].(bson.D)
		if !ok || d == nil || len(d) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		surveillance := bsonDToMap(d)

		// Fetch vehicle assigned to student
		vehID, ok := surveillance["vehicle_id"].(bson.ObjectID)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		vehicleDoc, err := s.getDoc(ctx, collVehicles, bson.M{"_id": vehID})
		if err != nil {
			return nil, nil, err
		}

		// Fetch routes assigned to student
		assignedRoutes, ok := surveillance["assigned_route_ids"].(bson.A)
		if !ok || len(assignedRoutes) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		var routeDocs []bson.M
		for _, r := range assignedRoutes {
			routeID, ok := r.(bson.ObjectID)
			if !ok {
				return nil, nil, myerrors.ErrMetaData
			}
			routeDoc, err := s.getDoc(ctx, collRoutes, bson.M{"_id": routeID})
			if err != nil {
				return nil, nil, err
			}
			routeDocs = append(routeDocs, routeDoc)
		}

		// replace ids with embedded documents
		surveillance["assigned_route_ids"] = routeDocs
		surveillance["vehicle_id"] = vehicleDoc
		studDoc["surveillance"] = surveillance
		students = append(students, studDoc)
	}
	superAcc[1].Value = students
	userDoc["super_account"] = superAcc
	ot.Val = val

	return userDoc, ot, nil
}

func (s *governanceService) fetchGuardianMetaData(ctx context.Context, userDoc bson.M) (bson.M, *auth.OtherClaims, error) {
	db := "metadata"
	collStudents := s.svcStore.Mongo.GetCollection(db, "students")
	collVehicles := s.svcStore.Mongo.GetCollection(db, "vehicles")
	collRoutes := s.svcStore.Mongo.GetCollection(db, "routes")

	val := struct {
		LinkedStudents    []string `json:"linked_students"`
		DelegatedStudents []string `json:"delegated_students"`
	}{
		DelegatedStudents: []string{},
	}
	ot := &auth.OtherClaims{
		Name: "parent_claims",
	}

	superAcc, ok := userDoc["super_account"].(bson.D)
	if !ok || len(superAcc) != 2 {
		return nil, nil, myerrors.ErrMissingMetaData
	}
	// superAcc[0] -> user_id
	if uuidStr, ok := uuidBinToString(superAcc[0].Value); ok {
		superAcc[0].Value = uuidStr
	}
	// superAcc[1] -> delegated_ids
	delegatedIDs, ok := superAcc[1].Value.(bson.A)
	if !ok || len(delegatedIDs) <= 0 {
		return nil, nil, myerrors.ErrMetaData
	}

	// Fetch all students delegated to the user
	var students []bson.M
	for _, l := range delegatedIDs {

		// Fetch student
		studUUID, ok := l.(bson.Binary)
		if !ok || studUUID.Subtype != 0x04 {
			return nil, nil, myerrors.ErrMetaData
		}
		studDoc, err := s.getDoc(ctx, collStudents, bson.M{"_id": studUUID})
		if err != nil {
			return nil, nil, err
		}

		// convert binary UUIDs to string
		if uuidStr, ok := uuidBinToString(studDoc["_id"]); ok {
			studDoc["_id"] = uuidStr
			val.DelegatedStudents = append(val.DelegatedStudents, uuidStr)
		}
		if uuidStr, ok := uuidBinToString(studDoc["parent_id"]); ok {
			studDoc["parent_id"] = uuidStr
		}

		// Check if student has opted for surveillance
		hasSurveillance, ok := studDoc["has_surveillance"].(bool)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		if !hasSurveillance {
			studDoc["surveillance"] = nil
			students = append(students, studDoc)
			continue
		}
		d, ok := studDoc["surveillance"].(bson.D)
		if !ok || d == nil || len(d) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		surveillance := bsonDToMap(d)

		// Fetch vehicle assigned to student
		vehID, ok := surveillance["vehicle_id"].(bson.ObjectID)
		if !ok {
			return nil, nil, myerrors.ErrMetaData
		}
		vehicleDoc, err := s.getDoc(ctx, collVehicles, bson.M{"_id": vehID})
		if err != nil {
			return nil, nil, err
		}

		// Fetch routes assigned to student
		assignedRoutes, ok := surveillance["assigned_route_ids"].(bson.A)
		if !ok || len(assignedRoutes) <= 0 {
			return nil, nil, myerrors.ErrMetaData
		}
		var routeDocs []bson.M
		for _, r := range assignedRoutes {
			routeID, ok := r.(bson.ObjectID)
			if !ok {
				return nil, nil, myerrors.ErrMetaData
			}
			routeDoc, err := s.getDoc(ctx, collRoutes, bson.M{"_id": routeID})
			if err != nil {
				return nil, nil, err
			}
			routeDocs = append(routeDocs, routeDoc)
		}

		// replace ids with embedded documents
		surveillance["assigned_route_ids"] = routeDocs
		surveillance["vehicle_id"] = vehicleDoc
		studDoc["surveillance"] = surveillance
		students = append(students, studDoc)
	}
	superAcc[1].Value = students
	userDoc["super_account"] = superAcc

	// convert binary UUID to string
	if uuidStr, ok := uuidBinToString(userDoc["_id"]); ok {
		userDoc["_id"] = uuidStr
	}

	return userDoc, ot, nil
}

func (s *governanceService) fetchSchoolAdminMetaData(ctx context.Context, userID uuid.UUID) (bson.M, error) {
	db := "metadata"
	collUsers := s.svcStore.Mongo.GetCollection(db, "users")
	collVehicles := s.svcStore.Mongo.GetCollection(db, "vehicles")
	collRoutes := s.svcStore.Mongo.GetCollection(db, "routes")

	id := uuidToBinary(userID)

	// Fetch user
	userDoc, err := s.getDoc(ctx, collUsers, bson.M{"_id": id})
	if err != nil {
		return nil, err
	}

	// check role exists
	role, ok := userDoc["role"].(string)
	if !ok || role == "" || role != string(auth.RoleSchoolAdmin) {
		return nil, myerrors.ErrMissingMetaData
	}

	// check account status
	if status, ok := userDoc["account_status"].(string); !ok || !strings.EqualFold(status, "active") {
		return nil, myerrors.ErrAccessDenied.WithInternalErr(errors.New("user account status is not set to active"))
	}

	schoolID, ok := userDoc["school_id"].(string)
	if !ok || schoolID == "" {
		return nil, myerrors.ErrMissingMetaData
	}

	// fetch all vehicles that have the referenced schoolID
	cursor, err := s.svcStore.Mongo.Find(ctx, collVehicles, bson.M{"school_id": schoolID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var vehicles []bson.M

	for cursor.Next(ctx) {
		var vehicleDoc bson.M
		if err := cursor.Decode(&vehicleDoc); err != nil {
			return nil, myerrors.ErrMetaData.WithInternalErr(err)
		}

		routeIDs, ok := vehicleDoc["route_ids"].(bson.A)
		if !ok || len(routeIDs) == 0 {
			return nil, myerrors.ErrMetaData
		}

		var routes []bson.M
		for _, rid := range routeIDs {
			objID, ok := rid.(bson.ObjectID)
			if !ok {
				return nil, myerrors.ErrMetaData
			}
			routeDoc, err := s.getDoc(ctx, collRoutes, bson.M{"_id": objID})
			if err != nil {
				return nil, err
			}
			routes = append(routes, routeDoc)
		}

		vehicleDoc["route_ids"] = routes
		vehicles = append(vehicles, vehicleDoc)
	}

	// convert user _id to string
	if uuidStr, ok := uuidBinToString(userDoc["_id"]); ok {
		userDoc["_id"] = uuidStr
	}

	userDoc["vehicles"] = vehicles
	return userDoc, nil
}
