package governance

import (
	"context"

	myerrors "github.com/Resolyte-official/service-governance/pkg/errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

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
	db := "resolyte"
	collUsers := s.mongoService.GetCollection(db, "users")
	collStudents := s.mongoService.GetCollection(db, "students")
	collVehicles := s.mongoService.GetCollection(db, "vehicles")
	collRoutes := s.mongoService.GetCollection(db, "routes")

	// Fetch user
	userDoc, err := s.getDoc(ctx, collUsers, bson.M{"_id": userID})
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
		studID, ok := l.(string)
		if !ok {
			return nil, myerrors.ErrMetaData
		}
		studDoc, err := s.getDoc(ctx, collStudents, bson.M{"_id": studID})
		if err != nil {
			return nil, err
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

	userDoc["linked_ids"] = students
	return &userDoc, nil
}
