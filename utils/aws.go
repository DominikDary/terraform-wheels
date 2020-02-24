package utils

import (
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/sts"
)

func IsAWSCredsOK() bool {
  svc := sts.New(session.New())
  input := &sts.GetCallerIdentityInput{}

  _, err := svc.GetCallerIdentity(input)
  if err != nil {
    return false
  }

  return true
}
