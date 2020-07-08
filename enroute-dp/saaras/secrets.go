// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2019 Saaras Inc.

package saaras

import (
	v1 "k8s.io/api/core/v1"
)

const qGetSecretsByOrgName string = `
query get_secret_by_org_name($oname: String!) {
  saaras_db_secret(where: {orgByOrgId: {org_name: {_eq: $oname}}}) {
    secret_id
    secret_name
    
    create_ts
    update_ts
    
    secret_artifactsBySecret_id {
      secret_artifact_id
      secret_artifact_name
      
      create_ts
      update_ts
      
      type
      key
      value
      
    }
  }
}
`

// TODO: Add org in query, right now the application_secret table is populated without org_id when populated through unify
const qGetSecretsByOrgName2 string = `
query
get_secrets_for_org_app($org_name: String!) {
  saaras_db_application_secret(where: {
    _and: [
		{org: {org_name: {_eq: $org_name}}}
  ]
  }) {
    
    secret_id
    secretsBySecretId {
		  secret_id
		  secret_name
      secret_artifactsBySecret_id {
        secret_artifact_id
        secret_artifact_name
        
        type
        key
        value
      }
    }
    
  }
}
`

const qGetSecretsByOrgName3 string = `
query
get_secrets_for_org_app($app_name: String!, $org_name: String!) {
  saaras_db_application_secret(where: {
    _and: [
    {applicationsByAppId: {app_name: {_eq: $app_name}}},
		{org: {org_name: {_eq: $org_name}}}
  ]
  }) {
    
    secret_id
    secretsBySecretId {
      secret_artifactsBySecret_id {
        secret_artifact_id
        secret_artifact_name
        
        type
        key
        value
      }
    }
    
  }
}
`

const qGetSecretsByOrgNameOutput string = `
DEBU[0394] secrets([]squeries.Secret) (len=1 cap=4) {
 (squeries.Secret) {
  Secret_artifactsBySecret_id: ([]squeries.SecretArtifact) (len=2 cap=4) {
   (squeries.SecretArtifact) {
    Secret_artifact_id: (string) (len=1) "8",
    Secret_artifact_name: (string) (len=9) "test1_key",
    Type: (string) (len=12) "tls_key_cert",
    Key: (string) (len=7) "tls_key",
    Value: (string) (len=1708) "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDcLHQKsH0y8udm\n9DMZG8FVTQnVDVI0JFwHXwcQQUUTO1LifddyVXc5dx3aLY2+1p6JZdNgZmic7IvY\nQwT8aBB3fH4lnityAlbWnK1yRQ6qIjPFBkzU3CH8l1jZaGMd39A2xWznZ1L3/pxy\nN6EJPSc77XqjDwGc40fsABETuU+GNRKtMvICZ2g9qGaDSGcdZgAlY+f2jysUB5Tr\nT6Du/rkd/yxHSaUZvCv/Wm0y2haNnSmbXz+PoorJSlzl8yBM+eaT6UYFy2SGVSFB\n9vPj7eSg1uck3lpjpseE3Pk4HoKiZlc44iZkU/YJ2ifjtmi85ZqqNfOQM/8YDwYN\n22SlpZ93AgMBAAECggEBAMgoVXoeRkNaFco/uHBcDh95ALZB/PhQEaXV2vsJCz3X\nkZs74fAcCF4iju34ucLDI68u9cHOd84pMVzyWIcKJ+YoNBoIt+BWhhFmsDuQ0isT\nGtNDzfc5BGC4SlqjDnBrNsOEKWQZR5ESU7F2JxzaDl/pnbK9Aq9Y49qFmQDAV65d\nwsglt7SBLHP6BFUxbrQOE+1OFk9Q57jDdbMPb1DiQIl/NFyyQpe0flvePjc9IXk9\nEDEz51OsHv81KmYjhNJR8oZiWbGTCNOx/ArqWuuK2D1yVTdF1LtzHrv9675J9TO3\nxd11ex38REQZ+GJbYfWJgW8g9WJSPC+uQrqtC5UbUhECgYEA/RPv0+oB+8Xg+9hq\nmUIz70rJ8+2opxK+vzzuihE46Lfe/Vt3Vc5TAl9jhcdPuN8DBxdEe7zJT37ufGaS\nnbExeT1NpRTcAqAkUxDBkjelvrXZkD7XaMv+mbjSLS/IOfqVXqG1kaFgms5TtsQC\nngz8ZUIPzKKEDvGbAat+MTK7EOkCgYEA3rdBkBAmr6KT2WiLy6iRLz1F/adSuOIQ\nJRI6Y/0NFEkQK26+Kj9hHC7QVvaeS7p1gKvH64hGgEUnndBganPYUlpWjX+WFHjC\nmPfOPuqd40GA57UNQ6oo4FeXYrrGyQUfK6yjuZvoeLG3d2rR++7tE2zFTc4K9983\n+dgrvCN38V8CgYEAkuY0qqRFXHiC3IzFa4pzDO4zhYSpBbmqwOTEbZ4Lk4HPTO7/\nuO3XXyQxZ6DGlL/WSRJnbQ+rJpq+IbWEW0ZUOlSsMiuGfXupOhIa2h209psl20Wu\n0aS/d0lBrnry1Tyv4UsqUosCwTkMfKUQA9/zzW7oLtcSon35hKGf0TzqOqkCgYEA\ny6XaB3cdSMBqXQPhwEnU59MparVTSMc9aAhw5/j9uqzMYkqTDGKD05di3gIH4MsQ\noqVw2wfzH1sczIs7fluLVFJSjnQ5sWJy3hjJuHIkCSdeTYEaLeMsGWc+gAK1vh42\n0GK+Gvxa5/HpBwLgG3PvyDFPgMOE9/5eWtC1vQTZqhUCgYBVndgdeZYAs5ToVDSD\n4xKOG90CDLZcsFa2zxlJedNsv8xKOt41FjZZnzdVad4GVsSLoMbDWe7spWd3+327\ntRYTlxKeJg6BMDsZB90UzL5/6SWpyYCKDV5lS/0NZXxuVyN3T3PCg4P+++L+Vhxn\nD7It/V+eWeoRGXVrjE04nlSGjA==\n-----END PRIVATE KEY-----\n"
   },
   (squeries.SecretArtifact) {
    Secret_artifact_id: (string) (len=1) "9",
    Secret_artifact_name: (string) (len=10) "test1_cert",
    Type: (string) (len=12) "tls_key_cert",
    Key: (string) (len=8) "tls_cert",
    Value: (string) (len=1289) "-----BEGIN CERTIFICATE-----\nMIIDizCCAnOgAwIBAgIJAMLoHtrOFjosMA0GCSqGSIb3DQEBCwUAMFsxCzAJBgNV\nBAYTAlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQww\nCgYDVQQKDANEaXMxFzAVBgNVBAMMDmluZ3Jlc3NwaXBlLmlvMCAXDTE5MDQyNDAz\nNTg0OFoYDzIxMTkwMzMxMDM1ODQ4WjBbMQswCQYDVQQGEwJVUzEPMA0GA1UECAwG\nRGVuaWFsMRQwEgYDVQQHDAtTcHJpbmdmaWVsZDEMMAoGA1UECgwDRGlzMRcwFQYD\nVQQDDA5pbmdyZXNzcGlwZS5pbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC\nggEBANwsdAqwfTLy52b0MxkbwVVNCdUNUjQkXAdfBxBBRRM7UuJ913JVdzl3Hdot\njb7Wnoll02BmaJzsi9hDBPxoEHd8fiWeK3ICVtacrXJFDqoiM8UGTNTcIfyXWNlo\nYx3f0DbFbOdnUvf+nHI3oQk9JzvteqMPAZzjR+wAERO5T4Y1Eq0y8gJnaD2oZoNI\nZx1mACVj5/aPKxQHlOtPoO7+uR3/LEdJpRm8K/9abTLaFo2dKZtfP4+iislKXOXz\nIEz55pPpRgXLZIZVIUH28+Pt5KDW5yTeWmOmx4Tc+TgegqJmVzjiJmRT9gnaJ+O2\naLzlmqo185Az/xgPBg3bZKWln3cCAwEAAaNQME4wHQYDVR0OBBYEFOOus5SS4KYz\n8qlscYkzMv3Wa4UJMB8GA1UdIwQYMBaAFOOus5SS4KYz8qlscYkzMv3Wa4UJMAwG\nA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAB3XDfIigVD4NVVu51yjFhWC\nn+Lwwq8v/sJz8hDq4Yeh9+v4NOgEq9QteHbQfU7QNkCNRBcP8hsS/65JN9sK2dZJ\nw54ulECCqeiz54yGE3Xj1nw8gejxfj8gV+OXvROkMKbFZoOVZObYL/iK+dyjCx/4\nAb3rddYbmj1f6ARRs4rns+6XhUv5wGiF9EFAY9hVfvn6la+QLrCWpumURPn+iLTi\n3ZielBcyLDMFnlB1tjrHb1Ou3o3ODgJ0ZcriTXJmnHrNpWSoIr4nSE9QToaAjDMP\nPXPgxFOstwN5MhxdJLkeBgP5A2OUxAppaJV69jpoY8PbPCkMMGsaBd4xKS1cd5k=\n-----END CERTIFICATE-----\n"
   }
  },
  Secret_name: (string) (len=5) "test1",
  Secret_id: (string) (len=2) "12"
 }
}  context=watch
`

const qGetSecretsByOrgNameOutput2 string = `
{
  "data": {
    "saaras_db_application_secret": [
      {
        "secretsBySecretId": {
          "secret_artifactsBySecret_id": [
            {
              "secret_artifact_id": "22",
              "secret_artifact_name": "test1_key",
              "type": "tls_key_cert",
              "key": "tls_key",
              "value": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDcLHQKsH0y8udm\n9DMZG8FVTQnVDVI0JFwHXwcQQUUTO1LifddyVXc5dx3aLY2+1p6JZdNgZmic7IvY\nQwT8aBB3fH4lnityAlbWnK1yRQ6qIjPFBkzU3CH8l1jZaGMd39A2xWznZ1L3/pxy\nN6EJPSc77XqjDwGc40fsABETuU+GNRKtMvICZ2g9qGaDSGcdZgAlY+f2jysUB5Tr\nT6Du/rkd/yxHSaUZvCv/Wm0y2haNnSmbXz+PoorJSlzl8yBM+eaT6UYFy2SGVSFB\n9vPj7eSg1uck3lpjpseE3Pk4HoKiZlc44iZkU/YJ2ifjtmi85ZqqNfOQM/8YDwYN\n22SlpZ93AgMBAAECggEBAMgoVXoeRkNaFco/uHBcDh95ALZB/PhQEaXV2vsJCz3X\nkZs74fAcCF4iju34ucLDI68u9cHOd84pMVzyWIcKJ+YoNBoIt+BWhhFmsDuQ0isT\nGtNDzfc5BGC4SlqjDnBrNsOEKWQZR5ESU7F2JxzaDl/pnbK9Aq9Y49qFmQDAV65d\nwsglt7SBLHP6BFUxbrQOE+1OFk9Q57jDdbMPb1DiQIl/NFyyQpe0flvePjc9IXk9\nEDEz51OsHv81KmYjhNJR8oZiWbGTCNOx/ArqWuuK2D1yVTdF1LtzHrv9675J9TO3\nxd11ex38REQZ+GJbYfWJgW8g9WJSPC+uQrqtC5UbUhECgYEA/RPv0+oB+8Xg+9hq\nmUIz70rJ8+2opxK+vzzuihE46Lfe/Vt3Vc5TAl9jhcdPuN8DBxdEe7zJT37ufGaS\nnbExeT1NpRTcAqAkUxDBkjelvrXZkD7XaMv+mbjSLS/IOfqVXqG1kaFgms5TtsQC\nngz8ZUIPzKKEDvGbAat+MTK7EOkCgYEA3rdBkBAmr6KT2WiLy6iRLz1F/adSuOIQ\nJRI6Y/0NFEkQK26+Kj9hHC7QVvaeS7p1gKvH64hGgEUnndBganPYUlpWjX+WFHjC\nmPfOPuqd40GA57UNQ6oo4FeXYrrGyQUfK6yjuZvoeLG3d2rR++7tE2zFTc4K9983\n+dgrvCN38V8CgYEAkuY0qqRFXHiC3IzFa4pzDO4zhYSpBbmqwOTEbZ4Lk4HPTO7/\nuO3XXyQxZ6DGlL/WSRJnbQ+rJpq+IbWEW0ZUOlSsMiuGfXupOhIa2h209psl20Wu\n0aS/d0lBrnry1Tyv4UsqUosCwTkMfKUQA9/zzW7oLtcSon35hKGf0TzqOqkCgYEA\ny6XaB3cdSMBqXQPhwEnU59MparVTSMc9aAhw5/j9uqzMYkqTDGKD05di3gIH4MsQ\noqVw2wfzH1sczIs7fluLVFJSjnQ5sWJy3hjJuHIkCSdeTYEaLeMsGWc+gAK1vh42\n0GK+Gvxa5/HpBwLgG3PvyDFPgMOE9/5eWtC1vQTZqhUCgYBVndgdeZYAs5ToVDSD\n4xKOG90CDLZcsFa2zxlJedNsv8xKOt41FjZZnzdVad4GVsSLoMbDWe7spWd3+327\ntRYTlxKeJg6BMDsZB90UzL5/6SWpyYCKDV5lS/0NZXxuVyN3T3PCg4P+++L+Vhxn\nD7It/V+eWeoRGXVrjE04nlSGjA==\n-----END PRIVATE KEY-----\n"
            },
            {
              "secret_artifact_id": "23",
              "secret_artifact_name": "test1_cert",
              "type": "tls_key_cert",
              "key": "tls_cert",
              "value": "-----BEGIN CERTIFICATE-----\nMIIDizCCAnOgAwIBAgIJAMLoHtrOFjosMA0GCSqGSIb3DQEBCwUAMFsxCzAJBgNV\nBAYTAlVTMQ8wDQYDVQQIDAZEZW5pYWwxFDASBgNVBAcMC1NwcmluZ2ZpZWxkMQww\nCgYDVQQKDANEaXMxFzAVBgNVBAMMDmluZ3Jlc3NwaXBlLmlvMCAXDTE5MDQyNDAz\nNTg0OFoYDzIxMTkwMzMxMDM1ODQ4WjBbMQswCQYDVQQGEwJVUzEPMA0GA1UECAwG\nRGVuaWFsMRQwEgYDVQQHDAtTcHJpbmdmaWVsZDEMMAoGA1UECgwDRGlzMRcwFQYD\nVQQDDA5pbmdyZXNzcGlwZS5pbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC\nggEBANwsdAqwfTLy52b0MxkbwVVNCdUNUjQkXAdfBxBBRRM7UuJ913JVdzl3Hdot\njb7Wnoll02BmaJzsi9hDBPxoEHd8fiWeK3ICVtacrXJFDqoiM8UGTNTcIfyXWNlo\nYx3f0DbFbOdnUvf+nHI3oQk9JzvteqMPAZzjR+wAERO5T4Y1Eq0y8gJnaD2oZoNI\nZx1mACVj5/aPKxQHlOtPoO7+uR3/LEdJpRm8K/9abTLaFo2dKZtfP4+iislKXOXz\nIEz55pPpRgXLZIZVIUH28+Pt5KDW5yTeWmOmx4Tc+TgegqJmVzjiJmRT9gnaJ+O2\naLzlmqo185Az/xgPBg3bZKWln3cCAwEAAaNQME4wHQYDVR0OBBYEFOOus5SS4KYz\n8qlscYkzMv3Wa4UJMB8GA1UdIwQYMBaAFOOus5SS4KYz8qlscYkzMv3Wa4UJMAwG\nA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAB3XDfIigVD4NVVu51yjFhWC\nn+Lwwq8v/sJz8hDq4Yeh9+v4NOgEq9QteHbQfU7QNkCNRBcP8hsS/65JN9sK2dZJ\nw54ulECCqeiz54yGE3Xj1nw8gejxfj8gV+OXvROkMKbFZoOVZObYL/iK+dyjCx/4\nAb3rddYbmj1f6ARRs4rns+6XhUv5wGiF9EFAY9hVfvn6la+QLrCWpumURPn+iLTi\n3ZielBcyLDMFnlB1tjrHb1Ou3o3ODgJ0ZcriTXJmnHrNpWSoIr4nSE9QToaAjDMP\nPXPgxFOstwN5MhxdJLkeBgP5A2OUxAppaJV69jpoY8PbPCkMMGsaBd4xKS1cd5k=\n-----END CERTIFICATE-----\n"
            }
          ]
        }
      }
    ]
  }
}
`

////////////////////////////  Secret //////////////////////////
type SecretArtifact struct {
	Secret_artifact_id   string
	Secret_artifact_name string
	Type                 string
	Key                  string
	Value                string
}

type SecretsBySId struct {
	Secret_id                   string
	Secret_name                 string
	Secret_artifactsBySecret_id []SecretArtifact
}

type Secret struct {
	SecretsBySecretId SecretsBySId
	Secret_id         string
}

type SaarasDbSecret struct {
	Saaras_db_application_secret []Secret
}

type DataPayloadSecrets struct {
	Data   SaarasDbSecret
	Errors []GraphErr
}

////////////////////////////  Secret //////////////////////////

func Saaras_secret__to__v1_secret(secrets *[]Secret) *[]v1.Secret {
	var v1secrets []v1.Secret
	for _, s := range *secrets {
		v1s := new(v1.Secret)
		v1s.Name = s.SecretsBySecretId.Secret_name
		// TODO
		v1s.Namespace = "trial_org_1"
		for _, artifact := range s.SecretsBySecretId.Secret_artifactsBySecret_id {
			// TODO: Use const define for string const "tls_key_cert"
			if artifact.Type == "tls_key_cert" {
				if v1s.Data == nil {
					v1s.Data = map[string][]byte{}
				}
				if artifact.Key == "tls_key" {
					v1s.Data[v1.TLSPrivateKeyKey] = []byte(artifact.Value)
				} else if artifact.Key == "tls_cert" {
					v1s.Data[v1.TLSCertKey] = []byte(artifact.Value)
				}
			}
		}

		v1secrets = append(v1secrets, *v1s)
	}

	return &v1secrets
}

//func Saaras_secret__to__v1_secret(secrets *[]Secret) *[]v1.Secret {
//	var v1secrets []v1.Secret
//	for _, s := range *secrets {
//		v1s := new(v1.Secret)
//		v1s.Name = s.Secret_name
//		// TODO
//		v1s.Namespace = "trial_org_1"
//		for _, artifact := range s.Secret_artifactsBySecret_id {
//			// TODO: Use const define for string const "tls_key_cert"
//			if artifact.Type == "tls_key_cert" {
//				if v1s.Data == nil {
//					v1s.Data = map[string][]byte{}
//				}
//				if artifact.Key == "tls_key" {
//					v1s.Data[v1.TLSPrivateKeyKey] = []byte(artifact.Value)
//				} else if artifact.Key == "tls_cert" {
//					v1s.Data[v1.TLSCertKey] = []byte(artifact.Value)
//				}
//			}
//		}
//
//		v1secrets = append(v1secrets, *v1s)
//	}
//
//	return &v1secrets
//}
