// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"strings"
	"time"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/productcatalogservice/genproto"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type productCatalog struct {
	catalog pb.ListProductsResponse
}

type contentAPIPrice struct {
	Value    string
	Currency string
}

type contentAPIProduct struct {
	OfferId      string
	Title        string
	Description  string
	ImageLink    string
	Price        contentAPIPrice
	ProductTypes []string
}

func (c *contentAPIProduct) String() string {
	return "offerId:" + c.OfferId
}

type contentAPIListProductsResponse struct {
	Resources []*contentAPIProduct
}

func (c *contentAPIListProductsResponse) String() string {
	outputStr := "["
	for i, p := range c.Resources {
		outputStr = outputStr + p.String()
		if i != len(c.Resources)-1 {
			outputStr = outputStr + ","
		}
	}
	return outputStr + "]"
}

func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)

	return &pb.ListProductsResponse{Products: p.parseCatalog()}, nil
}

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	time.Sleep(extraLatency)

	var found *pb.Product
	for i := 0; i < len(p.parseCatalog()); i++ {
		if req.Id == p.parseCatalog()[i].Id {
			found = p.parseCatalog()[i]
		}
	}

	if found == nil {
		return nil, status.Errorf(codes.NotFound, "no product with ID %s", req.Id)
	}
	return found, nil
}

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*pb.Product
	for _, product := range p.parseCatalog() {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}

func (p *productCatalog) parseCatalog() []*pb.Product {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		err := readProductsAPI(&p.catalog)
		if err != nil {
			return []*pb.Product{}
		}
	}

	return p.catalog.Products
}

func convertFromContentAPIProduct(cp *contentAPIProduct) *pb.Product {
	returnProduct := &pb.Product{
		Id:          cp.OfferId,
		Name:        cp.Title,
		Description: cp.Description,
		Picture:     cp.ImageLink,
	}
	priceDecimal, err := decimal.NewFromString(cp.Price.Value)
	if err != nil {
		log.Errorf("fail to create decimal from string with value %v", cp.Price.Value)
		return &pb.Product{}
	}
	returnProduct.PriceUsd = &pb.Money{
		Units: priceDecimal.IntPart(),
		Nanos: int32(priceDecimal.Sub(decimal.NewFromInt(priceDecimal.IntPart())).
			Mul(decimal.NewFromInt(1000_000_000)).IntPart()),
		CurrencyCode: cp.Price.Currency,
	}

	return returnProduct
}
