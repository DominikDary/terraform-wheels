package utils

import (
  "crypto/rand"
  "crypto/rsa"
  "crypto/x509"
  "encoding/pem"
  "fmt"
  "golang.org/x/crypto/ssh"
  "io/ioutil"
  // "log"
  "strings"
)

func CreatePublicRSAKeyFromPrivate(contents []byte, savePublicFileTo string) error {
  privateKey, err := parsePrivateKey(contents, "")
  if err != nil {
    return fmt.Errorf("Error parsing private key: %s", err.Error())
  }

  publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
  if err != nil {
    return fmt.Errorf("Error generating public key: %s", err.Error())
  }

  err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
  if err != nil {
    return fmt.Errorf("Error writing public key: %s", err.Error())
  }

  return nil
}

func CreateRSAKeyPair(savePrivateFileTo string, savePublicFileTo string) error {
  bitSize := 2048

  privateKey, err := generatePrivateKey(bitSize)
  if err != nil {
    return fmt.Errorf("Error generating private key: %s", err.Error())
  }

  publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
  if err != nil {
    return fmt.Errorf("Error generating public key: %s", err.Error())
  }

  privateKeyBytes := encodePrivateKeyToPEM(privateKey)

  err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
  if err != nil {
    return fmt.Errorf("Error writing private key: %s", err.Error())
  }

  err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
  if err != nil {
    return fmt.Errorf("Error writing public key: %s", err.Error())
  }

  return nil
}

func parsePrivateKey(contents []byte, rsaPrivateKeyPassword string) (*rsa.PrivateKey, error) {
  var err error

  privPem, _ := pem.Decode(contents)
  var privPemBytes []byte
  if privPem.Type != "RSA PRIVATE KEY" {
    return nil, fmt.Errorf("RSA private key is of the wrong type: %s", privPem.Type)
  }

  if rsaPrivateKeyPassword != "" {
    privPemBytes, err = x509.DecryptPEMBlock(privPem, []byte(rsaPrivateKeyPassword))
  } else {
    privPemBytes = privPem.Bytes
  }

  var parsedKey interface{}
  if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
    if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil { // note this returns type `interface{}`
      return nil, fmt.Errorf("Unable to parse RSA private key: %s", err.Error())
    }
  }

  var privateKey *rsa.PrivateKey
  var ok bool
  privateKey, ok = parsedKey.(*rsa.PrivateKey)
  if !ok {
    return nil, fmt.Errorf("This does not look like an RSA private key")
  }

  return privateKey, nil
}

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
  // Private Key generation
  privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
  if err != nil {
    return nil, err
  }

  // Validate Private Key
  err = privateKey.Validate()
  if err != nil {
    return nil, err
  }

  // log.Println("Private Key generated")
  return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
  // Get ASN.1 DER format
  privDER := x509.MarshalPKCS1PrivateKey(privateKey)

  // pem.Block
  privBlock := pem.Block{
    Type:    "RSA PRIVATE KEY",
    Headers: nil,
    Bytes:   privDER,
  }

  // Private key in PEM format
  privatePEM := pem.EncodeToMemory(&privBlock)

  return privatePEM
}

// generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// returns in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
  publicRsaKey, err := ssh.NewPublicKey(privatekey)
  if err != nil {
    return nil, err
  }

  pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

  // log.Println("Public key generated")
  return pubKeyBytes, nil
}

// writePemToFile writes keys to a file
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
  err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
  if err != nil {
    return err
  }

  // log.Printf("Key saved to: %s", saveFileTo)
  return nil
}

func GetPrivateKeyNameFromPublic(pubName string) string {
  if strings.Contains(pubName, "public") {
    return strings.ReplaceAll(pubName, "public", "private")
  }
  if strings.HasSuffix(pubName, ".pub") {
    return pubName[0 : len(pubName)-4]
  }
  if strings.Contains(pubName, "pub") {
    return strings.ReplaceAll(pubName, "pub", "priv")
  }

  return pubName + ".key"
}

func GetPublicKeyNameFromPrivate(pubName string) string {
  if strings.Contains(pubName, "private") {
    return strings.ReplaceAll(pubName, "private", "public")
  }
  if strings.HasSuffix(pubName, ".key") {
    return pubName[0:len(pubName)-4] + ".pub"
  }
  if strings.Contains(pubName, "priv") {
    return strings.ReplaceAll(pubName, "priv", "pub")
  }

  return pubName + ".pub"
}
