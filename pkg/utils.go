package governance

import (
	"context"

	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func uuidStrToBinary(u string) (bson.Binary, error) {
	parsed, err := uuid.Parse(u)
	if err != nil {
		return bson.Binary{}, err
	}
	return bson.Binary{Subtype: 0x04, Data: parsed[:]}, nil
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

func bsonDToMap(d bson.D) map[string]interface{} {
	m := make(map[string]interface{})
	for _, elem := range d {
		m[elem.Key] = elem.Value
	}
	return m
}

func (s *governanceService) getDoc(ctx context.Context, coll *mongo.Collection, filter any) (bson.M, error) {
	res, err := s.mongoService.FindOne(ctx, coll, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, myerrors.ErrMetaData.SetInternalErr(err)
		}
		return nil, err
	}
	var doc bson.M
	if err := res.Decode(&doc); err != nil {
		return nil, myerrors.ErrMongo.SetInternalErr(err)
	}
	return doc, nil
}

func (s *governanceService) fetchMetaData(ctx context.Context, userID string) (*bson.M, error) {
	db := "metadata"
	collUsers := s.mongoService.GetCollection(db, "users")
	collStudents := s.mongoService.GetCollection(db, "students")
	collVehicles := s.mongoService.GetCollection(db, "vehicles")
	collRoutes := s.mongoService.GetCollection(db, "routes")

	userUUID, err := uuidStrToBinary(userID)
	if err != nil {
		return nil, myerrors.ErrMetaData.SetInternalErr(err)
	}

	// Fetch user
	userDoc, err := s.getDoc(ctx, collUsers, bson.M{"_id": userUUID})
	if err != nil {
		return nil, err
	}

	// Fetch all students linked with the user
	studIDs, ok := userDoc["linked_ids"].(bson.A)
	if !ok || len(studIDs) <= 0 {
		return nil, myerrors.ErrMetaData
	}
	var students []bson.M
	for _, l := range studIDs {

		// Fetch student
		studUUID, ok := l.(bson.Binary)
		if !ok || studUUID.Subtype != 0x04 {
			return nil, myerrors.ErrMetaData
		}
		studDoc, err := s.getDoc(ctx, collStudents, bson.M{"_id": studUUID})
		if err != nil {
			return nil, err
		}

		// convert binary UUIDs to string
		if uuidStr, ok := uuidBinToString(studDoc["_id"]); ok {
			studDoc["_id"] = uuidStr
		}
		if uuidStr, ok := uuidBinToString(studDoc["parent_id"]); ok {
			studDoc["parent_id"] = uuidStr
		}

		// Check if student has opted for surveillance
		hasSurveillance, ok := studDoc["has_surveillance"].(bool)
		if !ok {
			return nil, myerrors.ErrMetaData
		}
		if !hasSurveillance {
			studDoc["surveillance"] = nil
			students = append(students, studDoc)
			continue
		}
		d, ok := studDoc["surveillance"].(bson.D)
		if !ok || d == nil || len(d) <= 0 {
			return nil, myerrors.ErrMetaData
		}
		surveillance := bsonDToMap(d)

		// Fetch vehicle assigned to student
		vehID, ok := surveillance["vehicle_id"].(bson.ObjectID)
		if !ok {
			return nil, myerrors.ErrMetaData
		}
		vehicleDoc, err := s.getDoc(ctx, collVehicles, bson.M{"_id": vehID})
		if err != nil {
			return nil, err
		}

		// Fetch routes assigned to student
		assignedRoutes, ok := surveillance["assigned_route_ids"].(bson.A)
		if !ok || len(assignedRoutes) <= 0 {
			return nil, myerrors.ErrMetaData
		}
		var routeDocs []bson.M
		for _, r := range assignedRoutes {
			routeID, ok := r.(bson.ObjectID)
			if !ok {
				return nil, myerrors.ErrMetaData
			}
			routeDoc, err := s.getDoc(ctx, collRoutes, bson.M{"_id": routeID})
			if err != nil {
				return nil, err
			}
			routeDocs = append(routeDocs, routeDoc)
		}

		// replace ids with embedded documents
		surveillance["assigned_route_ids"] = routeDocs
		surveillance["vehicle_id"] = vehicleDoc
		studDoc["surveillance"] = surveillance
		students = append(students, studDoc)
	}

	// convert binary UUID to string
	if uuidStr, ok := uuidBinToString(userDoc["_id"]); ok {
		userDoc["_id"] = uuidStr
	}

	userDoc["linked_ids"] = students
	return &userDoc, nil
}
