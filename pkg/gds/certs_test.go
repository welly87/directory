package gds_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/directory/pkg/gds"
	"github.com/trisacrypto/directory/pkg/gds/config"
	"github.com/trisacrypto/directory/pkg/gds/emails"
	"github.com/trisacrypto/directory/pkg/gds/models/v1"
	"github.com/trisacrypto/directory/pkg/sectigo"
	"github.com/trisacrypto/directory/pkg/sectigo/mock"
	pb "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"
	"google.golang.org/protobuf/proto"
)

// Test that the certificate manger correctly moves certificates across the request
// pipeline.
func (s *gdsTestSuite) TestCertManager() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	// Create a secret that the certificate manager can retrieve
	sm := s.svc.GetSecretManager().With(quebecCertReq.Id)
	ctx := context.Background()
	require.NoError(sm.CreateSecret(ctx, "password"))
	require.NoError(sm.AddSecretVersion(ctx, "password", []byte("qDhAwnfMjgDEzzUC")))

	// Let the certificate manager submit the certificate request
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP state should be changed to ISSUING_CERTIFICATE
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)
	// Audit log should contain one additional entry for ISSUING_CERTIFICATE
	log, err := models.GetAuditLog(v)
	require.NoError(err)
	require.Len(log, 5)
	require.Equal(pb.VerificationState_REVIEWED, log[4].PreviousState)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, log[4].CurrentState)
	require.Equal("automated", log[4].Source)

	// Certificate request should be updated
	certReq, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Greater(int(certReq.AuthorityId), 0)
	require.Greater(int(certReq.BatchId), 0)
	require.NotEmpty(certReq.BatchName)
	require.NotEmpty(certReq.BatchStatus)
	require.Greater(int(certReq.OrderNumber), 0)
	require.NotEmpty(certReq.CreationDate)
	require.NotEmpty(certReq.Profile)
	require.Empty(certReq.RejectReason)
	require.Equal(models.CertificateRequestState_PROCESSING, certReq.Status)
	// Audit log should contain one additional entry for PROCESSING
	require.Len(certReq.AuditLog, 3)
	require.Equal(models.CertificateRequestState_READY_TO_SUBMIT, certReq.AuditLog[2].PreviousState)
	require.Equal(models.CertificateRequestState_PROCESSING, certReq.AuditLog[2].CurrentState)
	require.Equal("automated", certReq.AuditLog[2].Source)

	// Let the certificate manager process the Sectigo response
	sent := time.Now()
	err = s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// Secret manager should contain the certificate
	secret, err := sm.GetLatestVersion(ctx, "cert")
	require.NoError(err)
	require.NotEmpty(secret)

	// VASP should contain the new certificate
	v, err = s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	idCert := v.IdentityCertificate
	require.NotNil(idCert)
	require.Greater(int(idCert.Version), 0)
	require.NotEmpty(idCert.SerialNumber)
	require.NotEmpty(idCert.Signature)
	require.NotEmpty(idCert.SignatureAlgorithm)
	require.NotEmpty(idCert.PublicKeyAlgorithm)
	require.NotNil(idCert.Subject)
	require.NotNil(idCert.Issuer)
	_, err = time.Parse(time.RFC3339, idCert.NotBefore)
	require.NoError(err)
	_, err = time.Parse(time.RFC3339, idCert.NotAfter)
	require.NoError(err)
	require.False(idCert.Revoked)
	require.NotEmpty(idCert.Data)
	require.NotEmpty(idCert.Chain)

	// VASP should contain the certificate ID in the extra
	certIDs, err := models.GetCertIDs(v)
	require.NoError(err)
	require.Len(certIDs, 1)
	require.NotEmpty(certIDs[0])

	// VASP state should be changed to VERIFIED
	require.Equal(pb.VerificationState_VERIFIED, v.VerificationStatus)
	// Audit log should contain one additional entry for VERIFIED
	log, err = models.GetAuditLog(v)
	require.NoError(err)
	require.Len(log, 6)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, log[5].PreviousState)
	require.Equal(pb.VerificationState_VERIFIED, log[5].CurrentState)
	require.Equal("automated", log[5].Source)

	// Certificate record should be created in the database
	cert, err := s.svc.GetStore().RetrieveCert(certIDs[0])
	require.NoError(err)
	require.Equal(certIDs[0], cert.Id)
	require.Equal(certReq.Id, cert.Request)
	require.Equal(v.Id, cert.Vasp)
	require.Equal(models.CertificateState_ISSUED, cert.Status)
	require.True(proto.Equal(idCert, cert.Details))

	// Email should be sent to one of the contacts
	messages := []*emailMeta{
		{
			contact:   v.Contacts.Legal,
			to:        v.Contacts.Legal.Email,
			from:      s.svc.GetConf().Email.ServiceEmail,
			subject:   emails.DeliverCertsRE,
			reason:    "deliver_certs",
			timestamp: sent,
		},
	}
	s.CheckEmails(messages)

	// Certificate request should be updated
	certReq, err = s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_COMPLETED, certReq.Status)
	require.Equal(cert.Id, certReq.Certificate)
	// Audit log should contain additional entries for DOWNLOADING, DOWNLOADED, and
	// COMPLETED
	require.Len(certReq.AuditLog, 6)
	require.Equal(models.CertificateRequestState_PROCESSING, certReq.AuditLog[3].PreviousState)
	require.Equal(models.CertificateRequestState_DOWNLOADING, certReq.AuditLog[3].CurrentState)
	require.Equal("automated", certReq.AuditLog[3].Source)
	require.Equal(models.CertificateRequestState_DOWNLOADING, certReq.AuditLog[4].PreviousState)
	require.Equal(models.CertificateRequestState_DOWNLOADED, certReq.AuditLog[4].CurrentState)
	require.Equal("automated", certReq.AuditLog[4].Source)
	require.Equal(models.CertificateRequestState_DOWNLOADED, certReq.AuditLog[5].PreviousState)
	require.Equal(models.CertificateRequestState_COMPLETED, certReq.AuditLog[5].CurrentState)
	require.Equal("automated", certReq.AuditLog[5].Source)
}

// Test that the certificate manager rejects requests when the VASP state is invalid.
func (s *gdsTestSuite) TestCertManagerBadState() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	defer s.loadReferenceFixtures()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	// Set VASP to pending review
	echoVASP.VerificationStatus = pb.VerificationState_PENDING_REVIEW
	require.NoError(s.svc.GetStore().UpdateVASP(echoVASP))

	// Run the cert manager for a loop
	require.NoError(s.svc.HandleCertificateRequests(certDir), "certman loop unsuccessful")

	// Certificate request should be rejected before submission
	certReq, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_CR_REJECTED, certReq.Status)

	// Set VASP to rejected
	echoVASP.VerificationStatus = pb.VerificationState_REJECTED
	require.NoError(s.svc.GetStore().UpdateVASP(echoVASP))

	// Run the cert manager for a loop
	require.NoError(s.svc.HandleCertificateRequests(certDir), "certman loop unsuccessful")

	// Certificate request should be rejected before submission
	certReq, err = s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_CR_REJECTED, certReq.Status)

	// Set VASP to verified for correct submission
	echoVASP.VerificationStatus = pb.VerificationState_VERIFIED
	require.NoError(s.svc.GetStore().UpdateVASP(echoVASP))
	quebecCertReq.Status = models.CertificateRequestState_READY_TO_SUBMIT
	require.NoError(s.svc.GetStore().UpdateCertReq(quebecCertReq))

	// Move the certificate to processing
	require.NoError(s.svc.HandleCertificateRequests(certDir), "certman loop unsuccessful")

	// Set VASP to rejected
	echoVASP.VerificationStatus = pb.VerificationState_REJECTED
	require.NoError(s.svc.GetStore().UpdateVASP(echoVASP))

	// Run the cert manager for a loop
	require.NoError(s.svc.HandleCertificateRequests(certDir), "certman loop unsuccessful")

	// Certificate request should be rejected before download
	certReq, err = s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_CR_REJECTED, certReq.Status)
	require.Empty(certReq.Certificate)
}

// Test that the certificate manager is able to process an end entity profile.
func (s *gdsTestSuite) TestCertManagerEndEntityProfile() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEndEntityCertificate)
	defer s.teardownCertManager()
	defer s.loadReferenceFixtures()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	quebecCertReq.Profile = sectigo.ProfileCipherTraceEndEntityCertificate
	quebecCertReq.Params = map[string]string{
		"organizationName":    "TRISA Member VASP",
		"localityName":        "Menlo Park",
		"stateOrProvinceName": "California",
		"countryName":         "US",
	}
	require.NoError(s.svc.GetStore().UpdateCertReq(quebecCertReq))

	// Create a secret that the certificate manager can retrieve.
	sm := s.svc.GetSecretManager().With(quebecCertReq.Id)
	ctx := context.Background()
	require.NoError(sm.CreateSecret(ctx, "password"))
	require.NoError(sm.AddSecretVersion(ctx, "password", []byte("qDhAwnfMjgDEzzUC")))

	// Run the certificate manager through two iterations to fully process the request.
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")
	err = s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should contain the new certificate
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_VERIFIED, v.VerificationStatus)
	require.NotNil(v.IdentityCertificate)

	// Certificate request should be updated
	cert, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_COMPLETED, cert.Status)
}

// Test that the certificate manager is able to process a CipherTraceEE profile.
func (s *gdsTestSuite) TestCertManagerCipherTraceEEProfile() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	defer s.loadReferenceFixtures()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	quebecCertReq.Profile = sectigo.ProfileCipherTraceEE
	require.NoError(s.svc.GetStore().UpdateCertReq(quebecCertReq))

	// Create a secret that the certificate manager can retrieve
	sm := s.svc.GetSecretManager().With(quebecCertReq.Id)
	ctx := context.Background()
	require.NoError(sm.CreateSecret(ctx, "password"))
	require.NoError(sm.AddSecretVersion(ctx, "password", []byte("qDhAwnfMjgDEzzUC")))

	// Run the certificate manager through two iterations to fully process the request.
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")
	err = s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should contain the new certificate
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_VERIFIED, v.VerificationStatus)
	require.NotNil(v.IdentityCertificate)

	// Certificate request should be updated
	cert, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_COMPLETED, cert.Status)
}

// Test that certificate submission fails if the user available balance is 0.
func (s *gdsTestSuite) TestSubmitNoBalance() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	mock.Handle(sectigo.AuthorityUserBalanceAvailableEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, 0)
	})

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	// Run the CertManager for a tick
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should still be in the ISSUING_CERTIFICATE state
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Cert request should still be in the READY_TO_SUBMIT state
	cert, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_READY_TO_SUBMIT, cert.Status)

	// Audit log should be updated
	log, err := models.GetAuditLog(v)
	require.NoError(err)
	require.Len(log, 5)
	require.Equal(pb.VerificationState_REVIEWED, log[4].PreviousState)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, log[4].CurrentState)
	require.Equal("automated", log[4].Source)
}

// Test that the certificate submission fails if there is no available password.
func (s *gdsTestSuite) TestSubmitNoPassword() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	// Run the CertManager for a tick
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should still be in the ISSUING_CERTIFICATE state
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Cert request should still be in the READY_TO_SUBMIT state
	cert, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_READY_TO_SUBMIT, cert.Status)

	// Audit log should be updated
	log, err := models.GetAuditLog(v)
	require.NoError(err)
	require.Len(log, 5)
	require.Equal(pb.VerificationState_REVIEWED, log[4].PreviousState)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, log[4].CurrentState)
	require.Equal("automated", log[4].Source)
}

// Test that the certificate submission fails if the batch request fails.
func (s *gdsTestSuite) TestSubmitBatchError() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEndEntityCertificate)
	defer s.teardownCertManager()
	defer s.loadReferenceFixtures()
	require := s.Require()

	echoVASP := s.fixtures[vasps]["echo"].(*pb.VASP)
	quebecCertReq := s.fixtures[certreqs]["quebec"].(*models.CertificateRequest)

	// Create a secret that the certificate manager can retrieve
	sm := s.svc.GetSecretManager().With(quebecCertReq.Id)
	ctx := context.Background()
	require.NoError(sm.CreateSecret(ctx, "password"))
	require.NoError(sm.AddSecretVersion(ctx, "password", []byte("qDhAwnfMjgDEzzUC")))

	// Certificate request with a missing country name
	quebecCertReq.Params = map[string]string{
		"organizationName":    "TRISA Member VASP",
		"localityName":        "Menlo Park",
		"stateOrProvinceName": "California",
	}
	require.NoError(s.svc.GetStore().UpdateCertReq(quebecCertReq))

	// Run the CertManager for a tick
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should still be in the ISSUING_CERTIFICATE state
	v, err := s.svc.GetStore().RetrieveVASP(echoVASP.Id)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Cert request should still be in the READY_TO_SUBMIT state
	cert, err := s.svc.GetStore().RetrieveCertReq(quebecCertReq.Id)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_READY_TO_SUBMIT, cert.Status)

	// Audit log should be updated
	log, err := models.GetAuditLog(v)
	require.NoError(err)
	require.Len(log, 5)
	require.Equal(pb.VerificationState_REVIEWED, log[4].PreviousState)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, log[4].CurrentState)
	require.Equal("automated", log[4].Source)
}

// Test that the certificate processing fails if the batch status request fails.
func (s *gdsTestSuite) TestProcessBatchDetailError() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	foxtrotId := s.fixtures[vasps]["foxtrot"].(*pb.VASP).Id

	// Batch detail returns an error
	mock.Handle(sectigo.BatchDetailEP, func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	// Run cert manager for one loop
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	v, err := s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Batch status can't be retrieved from both the detail and status endpoints.
	mock.Handle(sectigo.BatchDetailEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.BatchResponse{
			BatchID:      42,
			CreationDate: time.Now().Format(time.RFC3339),
		})
	})
	mock.Handle(sectigo.BatchStatusEP, func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	// Run cert manager for one loop
	err = s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	v, err = s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)
}

// Test that the certificate processing fails if there is still an active batch.
func (s *gdsTestSuite) TestProcessActiveBatch() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	foxtrotId := s.fixtures[vasps]["foxtrot"].(*pb.VASP).Id
	sierraId := s.fixtures[certreqs]["sierra"].(*models.CertificateRequest).Id

	// Batch detail returns an error
	mock.Handle(sectigo.BatchProcessingInfoEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.ProcessingInfoResponse{
			Active:  1,
			Success: 0,
			Failed:  0,
		})
	})

	// Run cert manager for one loop
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP should still be in the ISSUING_CERTIFICATE state
	v, err := s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Certificate request state should be changed to PROCESSING
	cert, err := s.svc.GetStore().RetrieveCertReq(sierraId)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.Status)

	// Audit log should be updated
	require.Len(cert.AuditLog, 4)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].PreviousState)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].CurrentState)
	require.Equal("automated", cert.AuditLog[2].Source)
}

// Test that the certificate processing fails if the batch request is rejected.
func (s *gdsTestSuite) TestProcessRejected() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	foxtrotId := s.fixtures[vasps]["foxtrot"].(*pb.VASP).Id
	sierraId := s.fixtures[certreqs]["sierra"].(*models.CertificateRequest).Id

	mock.Handle(sectigo.BatchDetailEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.BatchResponse{
			BatchID:      42,
			CreationDate: time.Now().Format(time.RFC3339),
			Status:       sectigo.BatchStatusRejected,
		})
	})
	mock.Handle(sectigo.BatchProcessingInfoEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.ProcessingInfoResponse{
			Active:  0,
			Success: 0,
			Failed:  1,
		})
	})

	// Run cert manager for one loop
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP state should be still be ISSUING_CERTIFICATE
	v, err := s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Certificate request state should be changed to CR_REJECTED
	cert, err := s.svc.GetStore().RetrieveCertReq(sierraId)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_CR_REJECTED, cert.Status)

	// Audit log should be updated
	require.Len(cert.AuditLog, 4)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].PreviousState)
	require.Equal(models.CertificateRequestState_CR_REJECTED, cert.AuditLog[3].CurrentState)
	require.Equal("automated", cert.AuditLog[3].Source)
}

// Test that the certificate processing fails if the batch request errors.
func (s *gdsTestSuite) TestProcessBatchError() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	foxtrotId := s.fixtures[vasps]["foxtrot"].(*pb.VASP).Id
	sierraId := s.fixtures[certreqs]["sierra"].(*models.CertificateRequest).Id

	mock.Handle(sectigo.BatchDetailEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.BatchResponse{
			BatchID:      42,
			CreationDate: time.Now().Format(time.RFC3339),
			Status:       sectigo.BatchStatusNotAcceptable,
		})
	})
	mock.Handle(sectigo.BatchProcessingInfoEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.ProcessingInfoResponse{
			Active:  0,
			Success: 0,
			Failed:  1,
		})
	})

	// Run cert manager for one loop
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP state should be still be ISSUING_CERTIFICATE
	v, err := s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Certificate request state should be changed to CR_ERRORED
	cert, err := s.svc.GetStore().RetrieveCertReq(sierraId)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_CR_ERRORED, cert.Status)

	// Audit log should be updated
	require.Len(cert.AuditLog, 4)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].PreviousState)
	require.Equal(models.CertificateRequestState_CR_ERRORED, cert.AuditLog[3].CurrentState)
	require.Equal("automated", cert.AuditLog[3].Source)
}

// Test that the certificate processing fails if the batch processing info request
// returns an unhandled sectigo state.
func (s *gdsTestSuite) TestProcessBatchNoSuccess() {
	certDir := s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	require := s.Require()

	foxtrotId := s.fixtures[vasps]["foxtrot"].(*pb.VASP).Id
	sierraId := s.fixtures[certreqs]["sierra"].(*models.CertificateRequest).Id

	mock.Handle(sectigo.BatchDetailEP, func(c *gin.Context) {
		c.JSON(http.StatusOK, &sectigo.BatchResponse{
			BatchID:      42,
			CreationDate: time.Now().Format(time.RFC3339),
			Status:       sectigo.BatchStatusNotAcceptable,
		})
	})

	// Run cert manager for one loop
	err := s.svc.HandleCertificateRequests(certDir)
	require.NoError(err, "certman loop unsuccessful")

	// VASP state should be still be ISSUING_CERTIFICATE
	v, err := s.svc.GetStore().RetrieveVASP(foxtrotId)
	require.NoError(err)
	require.Equal(pb.VerificationState_ISSUING_CERTIFICATE, v.VerificationStatus)

	// Certificate request state should be changed to PROCESSING
	cert, err := s.svc.GetStore().RetrieveCertReq(sierraId)
	require.NoError(err)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.Status)

	// Audit log should be updated
	require.Len(cert.AuditLog, 4)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].PreviousState)
	require.Equal(models.CertificateRequestState_PROCESSING, cert.AuditLog[3].CurrentState)
	require.Equal("automated", cert.AuditLog[3].Source)
}

func (s *gdsTestSuite) TestCertManagerLoop() {
	s.setupCertManager(sectigo.ProfileCipherTraceEE)
	defer s.teardownCertManager()
	s.runCertManager(s.svc.GetConf().CertMan.Interval)
}

func (s *gdsTestSuite) setupCertManager(profile string) (certDir string) {
	require := s.Require()
	certDir, err := ioutil.TempDir("testdata", "certs-*")
	require.NoError(err)
	conf := gds.MockConfig()
	conf.Sectigo.Profile = profile
	conf.CertMan = config.CertManConfig{
		Interval: time.Millisecond,
		Storage:  certDir,
	}
	require.NoError(os.MkdirAll(conf.CertMan.Storage, 0755))
	s.SetConfig(conf)
	s.LoadFullFixtures()
	return certDir
}

func (s *gdsTestSuite) teardownCertManager() {
	s.ResetConfig()
	s.ResetFixtures()
	emails.PurgeMockEmails()
	os.RemoveAll(s.svc.GetConf().CertMan.Storage)
}

// Helper function that spins up the CertificateManager for the specified duration,
// sends the stop signal, and waits for it to finish.
func (s *gdsTestSuite) runCertManager(interval time.Duration) {
	// Start the certificate manager
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.svc.CertManager(stop)
	}()

	// Wait for the interval to elapse
	time.Sleep(interval)

	// Make sure that the certificate manager is stopped before we proceed
	close(stop)
	wg.Wait()
}
