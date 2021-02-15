package olm

import (
	"fmt"
	"time"

	"github.com/openshift-pipelines/release-tests/pkg/assert"
	"github.com/openshift-pipelines/release-tests/pkg/clients"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Interval specifies the time between two polls.
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout            = 5 * time.Minute
	OperatorsNamespace = "openshift-operators"
	OLMNamespace       = "openshift-marketplace"
)

var (
	immediate             = int64(0)
	immediateDeleteOption = &metav1.DeleteOptions{GracePeriodSeconds: &immediate}
)

// Subscription helps you to subscribe openshift-pipelines-operator-rh
func Subscription(subscriptionName, channel, catalogsource string) *v1alpha1.Subscription {
	//namespace, name, catalogSourceName, packageName, channel string, approval v1alpha1.Approval
	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.SubscriptionKind,
			APIVersion: v1alpha1.SubscriptionCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: OperatorsNamespace,
			Name:      subscriptionName,
		},
		Spec: &v1alpha1.SubscriptionSpec{
			CatalogSource:          catalogsource,
			CatalogSourceNamespace: OLMNamespace,
			Package:                subscriptionName,
			Channel:                channel,
			InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
		},
	}
}

func SubscribeAndWaitForOperatorToBeReady(cs *clients.Clients, subscriptionName, channel, catalogsource string) (*v1alpha1.Subscription, error) {
	if _, err := createSubscription(cs, subscriptionName, channel, catalogsource); err != nil {
		return nil, err
	}

	subs, err := WaitForSubscriptionState(cs, subscriptionName, OperatorsNamespace, IsSubscriptionInstalledCSVPresent)
	if err != nil {
		return nil, err
	}

	csvName := subs.Status.InstalledCSV

	_, err = WaitForClusterServiceVersionState(cs, csvName, OperatorsNamespace, IsCSVSucceeded)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

func UptadeSubscriptionAndWaitForOperatorToBeReady(cs *clients.Clients, subscriptionName, channel string) (*v1alpha1.Subscription, error) {
	if _, err := UpdateSubscription(cs, subscriptionName, channel); err != nil {
		return nil, err
	}

	subs, err := WaitForSubscriptionState(cs, subscriptionName, OperatorsNamespace, IsSubscriptionInstalledCSVPresent)
	if err != nil {
		return nil, err
	}

	csvName := subs.Status.InstalledCSV

	_, err = WaitForClusterServiceVersionState(cs, csvName, OperatorsNamespace, IsCSVSucceeded)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

func getSubcription(cs *clients.Clients, name string) *v1alpha1.Subscription {
	subscription, err := cs.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Get(name, metav1.GetOptions{})
	assert.NoError(err, fmt.Sprintf("Unable to retrive Subscription: [%s] from namespace [%s]\n", name, OperatorsNamespace))
	return subscription
}

func createSubscription(cs *clients.Clients, name, channel, catalogsource string) (*v1alpha1.Subscription, error) {
	subs, err := cs.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Create(Subscription(name, channel, catalogsource))
	if err != nil {
		return nil, err
	}
	return subs, nil
}

// OperatorCleanup deletes All related CSVs, subscription & installplan
func OperatorCleanup(cs *clients.Clients, name string) {
	sub := getSubcription(cs, name)
	// Delete CSV
	err := cs.OLM.OperatorsV1alpha1().ClusterServiceVersions(OperatorsNamespace).DeleteCollection(immediateDeleteOption,
		metav1.ListOptions{})
	if err != nil {
		assert.NoError(err, fmt.Sprintf("failed deleting CSVs in Namespace: %s, %v", OperatorsNamespace, err))
	}

	// DeleteInstallPlan
	err = cs.OLM.OperatorsV1alpha1().InstallPlans(OperatorsNamespace).Delete(sub.Status.Install.Name, immediateDeleteOption)
	if err != nil {
		assert.NoError(err, fmt.Sprintf("failed deleting InstallPlan [%s] in Namespace: %s, %v", sub.Status.Install.Name, OperatorsNamespace, err))
	}

	// Unsubscribe
	err = cs.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).DeleteCollection(immediateDeleteOption, metav1.ListOptions{})
	if err != nil {
		assert.NoError(err, fmt.Sprintf("failed to clean all Subscriptions in %s, %v", OperatorsNamespace, err))
	}
}

func UpdateSubscription(cs *clients.Clients, name, channel string) (*v1alpha1.Subscription, error) {
	subscription := getSubcription(cs, name)
	subscription.Spec.Channel = channel
	subs, err := cs.OLM.OperatorsV1alpha1().Subscriptions(OperatorsNamespace).Update(subscription)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func WaitForSubscriptionState(cs *clients.Clients, name, namespace string, inState func(s *v1alpha1.Subscription, err error) (bool, error)) (*v1alpha1.Subscription, error) {
	var lastState *v1alpha1.Subscription
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = cs.OLM.OperatorsV1alpha1().Subscriptions(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "subscription %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func WaitForClusterServiceVersionState(cs *clients.Clients, name, namespace string, inState func(s *v1alpha1.ClusterServiceVersion, err error) (bool, error)) (*v1alpha1.ClusterServiceVersion, error) {
	var lastState *v1alpha1.ClusterServiceVersion
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = cs.OLM.OperatorsV1alpha1().ClusterServiceVersions(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "clusterserviceversion %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func IsCSVSucceeded(c *v1alpha1.ClusterServiceVersion, err error) (bool, error) {
	return c.Status.Phase == "Succeeded", err
}

func IsSubscriptionInstalledCSVPresent(s *v1alpha1.Subscription, err error) (bool, error) {
	return s.Status.InstalledCSV != "" && s.Status.InstalledCSV != "<none>", err
}
