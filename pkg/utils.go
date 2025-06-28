package governance

import (
	"context"

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

func (s *governanceService) fetchMetaData(ctx context.Context, userID uuid.UUID) (bson.M, error) {
	db := "metadata"
	collUsers := s.svcStore.Mongo.GetCollection(db, "users")
	collStudents := s.svcStore.Mongo.GetCollection(db, "students")
	collVehicles := s.svcStore.Mongo.GetCollection(db, "vehicles")
	collRoutes := s.svcStore.Mongo.GetCollection(db, "routes")

	// Fetch user
	userDoc, err := s.getDoc(ctx, collUsers, bson.M{"_id": uuidToBinary(userID)})
	if err != nil {
		return nil, err
	}

	// check role exists
	role, ok := userDoc["role"].(string)
	if !ok || role == "" {
		return nil, myerrors.ErrMissingMetaData
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
	return userDoc, nil
}
