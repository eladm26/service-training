package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ardanlabs/service/business/api/data/migrate"
	"github.com/ardanlabs/service/business/api/data/sqldb"
	"github.com/golang-jwt/jwt/v4"
	"github.com/open-policy-agent/opa/v1/rego"
)

func main() {
	err := Migrate()
	if err != nil {
		log.Fatalln(err)
	}

}

//go:embed rego/authentication.rego
var opaAuthentication string

// GenerateToken generates a signed JWT token string representing the user Claims.
func GenToken() (string, error) {
	// Generating a token requires defining a set of claims. In this applications
	// case, we only care about defining the subject and the user in question and
	// the roles they have on the database. This token will expire in a year.
	//
	// iss (issuer): Issuer of the JWT
	// sub (subject): Subject of the JWT (the user)
	// aud (audience): Recipient for which the JWT is intended
	// exp (expiration time): Time after which the JWT expires
	// nbf (not before time): Time before which the JWT must not be accepted for processing
	// iat (issued at time): Time at which the JWT was issued; can be used to determine age of the JWT
	// jti (JWT ID): Unique identifier; can be used to prevent the JWT from being replayed (allows a token to be used only once)
	claims := struct {
		jwt.RegisteredClaims
		Roles []string
	}{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "026f30a8-f048-4822-87e3-39bcf0e2353f",
			Issuer:    "service project",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(8760 * time.Hour)),
		},
		Roles: []string{"USER"},
	}

	method := jwt.GetSigningMethod(jwt.SigningMethodRS256.Name)
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = "54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"

	privateKeyPEM, err := os.ReadFile("zarf/keys/54bb2165-71e1-41a6-af3e-7da4a0e1e2c1.pem")
	if err != nil {
		return "", fmt.Errorf("reading private pem: %w", err)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("parsing private pem: %w", err)
	}

	str, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	fmt.Printf("------BEGIN TOKEN------\n%s\n------END TOKEN------\n\n", str)

	// -------------------------------------------------------------------------

	// Marshal the public key from the private key to PKIK
	asn1Bytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("marshaling public key: %w", err)
	}

	// Construct the PEM block for the puvblic key
	publicBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	// Wrtie the public key to the public key file
	if err := pem.Encode(os.Stdout, &publicBlock); err != nil {
		return "", fmt.Errorf("encoding to public file: %w", err)
	}

	var b bytes.Buffer
	if err := pem.Encode(&b, &publicBlock); err != nil {
		return "", fmt.Errorf("encoding to public file: %w", err)
	}

	// -------------------------------------------------------------------------

	ctx := context.Background()
	query := fmt.Sprintf("x = data.%s.%s", "ardan.rego", "auth")

	q, err := rego.New(
		rego.Query(query),
		rego.Module("policy.rego", opaAuthentication),
	).PrepareForEval(ctx)
	if err != nil {
		return "", err
	}

	input := map[string]any{
		"Key":   b.String(),
		"Token": str,
		"ISS":   "service project",
	}

	results, err := q.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return "", fmt.Errorf("query: %w", err)
	}

	if len(results) == 0 {
		return "", errors.New("No results")
	}

	result, ok := results[0].Bindings["x"].(bool)
	if !ok || !result {
		return "", fmt.Errorf("bindings results[%v] ok[%v]", result, ok)
	}

	fmt.Println("TOKEN VALIDATED")

	return str, nil

}

// GenKey creates an x509 private/public key for auth tokens.
func GenKey() error {
	// Generate a new private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	// Create a file for the private key information in PEM form
	privateFile, err := os.Create("private.pem")
	if err != nil {
		return fmt.Errorf("creating private file: %w", err)
	}
	defer privateFile.Close()

	// Construct a PEM block
	privateBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write the private key to the private key file
	if err := pem.Encode(privateFile, &privateBlock); err != nil {
		return fmt.Errorf("encoding to private file: %w", err)
	}

	// Create file for the public key information in PEM form
	publicFile, err := os.Create("public.pem")
	if err != nil {
		return fmt.Errorf("creating public file: %w", err)
	}
	defer publicFile.Close()

	// Marshal the public key from the private key to PKIK
	asn1Bytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("marshaling public key: %w", err)
	}

	// Construct the PEM block for the puvblic key
	publicBlock := pem.Block{
		Type:  "PUBLIC_KEY",
		Bytes: asn1Bytes,
	}

	// Wrtie the public key to the public key file
	if err := pem.Encode(publicFile, &publicBlock); err != nil {
		return fmt.Errorf("encoding to public file: %w", err)
	}

	fmt.Println("private and public key files generated")

	return nil
}

// Migrate creates the schema in the database
func Migrate() error {
	dbConfig := sqldb.Config{
		User:         "postgres",
		Password:     "postgres",
		HostPort:     "database-service.sales-system.svc.cluster.local",
		Name:         "postgres",
		MaxIdleConns: 2,
		MaxOpenConns: 0,
		DisableTLS:   true,
	}

	db, err := sqldb.Open(dbConfig)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := migrate.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	fmt.Println("migration complete")

	if err := migrate.Seed(ctx, db); err != nil {
		return fmt.Errorf("seed database: %w", err)
	}

	fmt.Println("seed data complete")
	return nil
}
