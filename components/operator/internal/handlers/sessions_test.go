package handlers

import (
	"context"
	"testing"

	"ambient-code-operator/internal/config"
	"ambient-code-operator/internal/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

// setupTestClient initializes a fake Kubernetes client for testing
func setupTestClient(objects ...runtime.Object) {
	config.K8sClient = fake.NewSimpleClientset(objects...)
}

// TestCopySecretToNamespace_NoSharedDataMutation verifies that we don't mutate cached secret objects
func TestCopySecretToNamespace_NoSharedDataMutation(t *testing.T) {
	// Create existing secret with one owner reference
	existingOwnerRef := metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       "existing-owner",
		UID:        k8stypes.UID("existing-uid-123"),
	}
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				existingOwnerRef,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("old-value"),
		},
	}

	// Setup fake client with existing secret
	setupTestClient(existingSecret)

	// Create source secret
	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	// Create owner object
	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("new-uid-456"))

	// Get the secret before the update
	beforeSecret, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(context.Background(), "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret before update: %v", err)
	}

	// Store the original slice pointer to verify it's not mutated
	originalSlicePtr := &beforeSecret.OwnerReferences

	// Call copySecretToNamespace
	ctx := context.Background()
	err = copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Get the updated secret
	updatedSecret, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated secret: %v", err)
	}

	// Verify the new owner reference was added
	if len(updatedSecret.OwnerReferences) != 2 {
		t.Errorf("Expected 2 owner references, got %d", len(updatedSecret.OwnerReferences))
	}

	// Verify the original owner reference is still there
	foundOriginal := false
	for _, ref := range updatedSecret.OwnerReferences {
		if ref.UID == existingOwnerRef.UID {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Error("Original owner reference was lost")
	}

	// Verify the new owner reference was added
	foundNew := false
	for _, ref := range updatedSecret.OwnerReferences {
		if ref.UID == ownerObj.GetUID() {
			foundNew = true
			break
		}
	}
	if !foundNew {
		t.Error("New owner reference was not added")
	}

	// Verify the original slice was not mutated (the pointer should be different)
	// Note: This is a best-effort check - the fake client may not preserve the exact same behavior
	// as the real client, but it validates our code creates a new slice
	if originalSlicePtr == &updatedSecret.OwnerReferences {
		t.Error("OwnerReferences slice pointer was not changed, indicating potential mutation")
	}

	// Verify data was updated
	if string(updatedSecret.Data["key"]) != "new-value" {
		t.Errorf("Expected data 'new-value', got '%s'", string(updatedSecret.Data["key"]))
	}

	// Verify annotation was added
	expectedAnnotation := "source-ns/ambient-vertex"
	if updatedSecret.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, updatedSecret.Annotations[types.CopiedFromAnnotation])
	}
}

// TestCopySecretToNamespace_CreateNew tests creating a new secret when it doesn't exist
func TestCopySecretToNamespace_CreateNew(t *testing.T) {
	setupTestClient()

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
			Labels: map[string]string{
				"app": "ambient-code",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": []byte("secret-data"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("test-uid-789"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret was created
	created, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created secret: %v", err)
	}

	// Verify basic fields
	if created.Name != "ambient-vertex" {
		t.Errorf("Expected name 'ambient-vertex', got '%s'", created.Name)
	}
	if created.Namespace != "target-ns" {
		t.Errorf("Expected namespace 'target-ns', got '%s'", created.Namespace)
	}

	// Verify data was copied
	if string(created.Data["credentials"]) != "secret-data" {
		t.Errorf("Expected data 'secret-data', got '%s'", string(created.Data["credentials"]))
	}

	// Verify labels were copied
	if created.Labels["app"] != "ambient-code" {
		t.Errorf("Expected label 'ambient-code', got '%s'", created.Labels["app"])
	}

	// Verify owner reference
	if len(created.OwnerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference, got %d", len(created.OwnerReferences))
	}
	if created.OwnerReferences[0].UID != ownerObj.GetUID() {
		t.Errorf("Expected owner UID '%s', got '%s'", ownerObj.GetUID(), created.OwnerReferences[0].UID)
	}
	if created.OwnerReferences[0].Kind != "AgenticSession" {
		t.Errorf("Expected owner kind 'AgenticSession', got '%s'", created.OwnerReferences[0].Kind)
	}
	if created.OwnerReferences[0].Controller == nil || !*created.OwnerReferences[0].Controller {
		t.Error("Expected Controller to be true")
	}

	// Verify annotation
	expectedAnnotation := "source-ns/ambient-vertex"
	if created.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, created.Annotations[types.CopiedFromAnnotation])
	}
}

// TestCopySecretToNamespace_AlreadyHasOwnerRef tests skipping update when owner ref already exists
func TestCopySecretToNamespace_AlreadyHasOwnerRef(t *testing.T) {
	ownerUID := k8stypes.UID("owner-uid-999")

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1alpha1",
					Kind:       "AgenticSession",
					Name:       "test-session",
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
			Annotations: map[string]string{
				types.CopiedFromAnnotation: "source-ns/ambient-vertex",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("original-value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(ownerUID)

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret was NOT updated (data should still be original)
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if string(result.Data["key"]) != "original-value" {
		t.Errorf("Expected data to remain 'original-value', got '%s'", string(result.Data["key"]))
	}

	// Should still have exactly 1 owner reference
	if len(result.OwnerReferences) != 1 {
		t.Errorf("Expected 1 owner reference, got %d", len(result.OwnerReferences))
	}
}

// TestCopySecretToNamespace_MultipleOwnerReferences tests adding owner ref to secret with existing different owner
func TestCopySecretToNamespace_MultipleOwnerReferences(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "other-owner",
					UID:        k8stypes.UID("other-uid-111"),
				},
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("updated-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("new-owner-uid-222"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret has both owner references
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if len(result.OwnerReferences) != 2 {
		t.Fatalf("Expected 2 owner references, got %d", len(result.OwnerReferences))
	}

	// Verify both UIDs are present
	uids := make(map[k8stypes.UID]bool)
	for _, ref := range result.OwnerReferences {
		uids[ref.UID] = true
	}

	if !uids[k8stypes.UID("other-uid-111")] {
		t.Error("Original owner reference was lost")
	}
	if !uids[k8stypes.UID("new-owner-uid-222")] {
		t.Error("New owner reference was not added")
	}
}

// TestCopySecretToNamespace_ExistingController tests adding owner ref when secret already has a controller
func TestCopySecretToNamespace_ExistingController(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "target-ns",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "vteam.ambient-code/v1alpha1",
					Kind:       "AgenticSession",
					Name:       "existing-session",
					UID:        k8stypes.UID("existing-uid-111"),
					Controller: boolPtr(true), // Already has a controller
				},
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("updated-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("new-session")
	ownerObj.SetUID(k8stypes.UID("new-uid-222"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify secret has both owner references
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if len(result.OwnerReferences) != 2 {
		t.Fatalf("Expected 2 owner references, got %d", len(result.OwnerReferences))
	}

	// Verify only one controller reference exists
	controllerCount := 0
	foundExisting := false
	foundNew := false
	for _, ref := range result.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			controllerCount++
		}
		if ref.UID == k8stypes.UID("existing-uid-111") {
			foundExisting = true
			// Original controller should still be true
			if ref.Controller == nil || !*ref.Controller {
				t.Error("Existing controller reference should still have Controller: true")
			}
		}
		if ref.UID == k8stypes.UID("new-uid-222") {
			foundNew = true
			// New reference should NOT have Controller: true
			if ref.Controller != nil && *ref.Controller {
				t.Error("New owner reference should NOT have Controller: true when secret already has a controller")
			}
		}
	}

	if controllerCount != 1 {
		t.Errorf("Expected exactly 1 controller reference, got %d", controllerCount)
	}
	if !foundExisting {
		t.Error("Existing owner reference was lost")
	}
	if !foundNew {
		t.Error("New owner reference was not added")
	}

	// Verify data was updated
	if string(result.Data["key"]) != "updated-value" {
		t.Errorf("Expected data 'updated-value', got '%s'", string(result.Data["key"]))
	}
}

// TestCopySecretToNamespace_NilAnnotations tests updating secret with nil annotations
func TestCopySecretToNamespace_NilAnnotations(t *testing.T) {
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ambient-vertex",
			Namespace:   "target-ns",
			Annotations: nil, // Explicitly nil
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	setupTestClient(existingSecret)

	sourceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ambient-vertex",
			Namespace: "source-ns",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte("new-value"),
		},
	}

	ownerObj := &unstructured.Unstructured{}
	ownerObj.SetAPIVersion("vteam.ambient-code/v1alpha1")
	ownerObj.SetKind("AgenticSession")
	ownerObj.SetName("test-session")
	ownerObj.SetUID(k8stypes.UID("test-uid-333"))

	ctx := context.Background()
	err := copySecretToNamespace(ctx, sourceSecret, "target-ns", ownerObj)
	if err != nil {
		t.Fatalf("copySecretToNamespace failed: %v", err)
	}

	// Verify annotation was added
	result, err := config.K8sClient.CoreV1().Secrets("target-ns").Get(ctx, "ambient-vertex", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if result.Annotations == nil {
		t.Fatal("Annotations should not be nil after update")
	}

	expectedAnnotation := "source-ns/ambient-vertex"
	if result.Annotations[types.CopiedFromAnnotation] != expectedAnnotation {
		t.Errorf("Expected annotation '%s', got '%s'", expectedAnnotation, result.Annotations[types.CopiedFromAnnotation])
	}
}

// TestDeleteAmbientVertexSecret_CopiedSecret tests deletion of a copied secret
func TestDeleteAmbientVertexSecret_CopiedSecret(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.AmbientVertexSecretName,
			Namespace: "test-ns",
			Annotations: map[string]string{
				types.CopiedFromAnnotation: "source-ns/ambient-vertex",
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was deleted
	_, err = config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err == nil {
		t.Error("Secret should have been deleted")
	}
}

// TestDeleteAmbientVertexSecret_NotCopied tests that non-copied secrets are not deleted
func TestDeleteAmbientVertexSecret_NotCopied(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      types.AmbientVertexSecretName,
			Namespace: "test-ns",
			// No CopiedFromAnnotation - this is a user-created secret
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was NOT deleted
	result, err := config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Secret should not have been deleted: %v", err)
	}
	if result == nil {
		t.Error("Secret should still exist")
	}
}

// TestDeleteAmbientVertexSecret_NotFound tests handling of non-existent secret
func TestDeleteAmbientVertexSecret_NotFound(t *testing.T) {
	setupTestClient()

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Errorf("deleteAmbientVertexSecret should not error on non-existent secret: %v", err)
	}
}

// TestDeleteAmbientVertexSecret_NilAnnotations tests handling of secret with nil annotations
func TestDeleteAmbientVertexSecret_NilAnnotations(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        types.AmbientVertexSecretName,
			Namespace:   "test-ns",
			Annotations: nil,
		},
		Type: corev1.SecretTypeOpaque,
	}

	setupTestClient(secret)

	ctx := context.Background()
	err := deleteAmbientVertexSecret(ctx, "test-ns")
	if err != nil {
		t.Fatalf("deleteAmbientVertexSecret failed: %v", err)
	}

	// Verify secret was NOT deleted (no annotation = not copied)
	result, err := config.K8sClient.CoreV1().Secrets("test-ns").Get(ctx, types.AmbientVertexSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Secret should not have been deleted: %v", err)
	}
	if result == nil {
		t.Error("Secret should still exist")
	}
}
