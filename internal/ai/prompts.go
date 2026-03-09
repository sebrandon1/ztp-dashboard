package ai

import "fmt"

func ProvisioningErrorPrompt(clusterName string, conditions interface{}) string {
	return fmt.Sprintf(`You are an expert OpenShift ZTP (Zero Touch Provisioning) engineer. Analyze the following provisioning error for cluster "%s" and provide:

1. A clear explanation of what went wrong
2. The most likely root cause
3. Step-by-step remediation instructions
4. Any relevant documentation references

Cluster conditions:
%v

Provide your response in clear markdown format.`, clusterName, conditions)
}

func ClusterHealthPrompt(clusterName string, healthData interface{}) string {
	return fmt.Sprintf(`You are an expert OpenShift cluster health analyst. Analyze the health status of spoke cluster "%s" and provide:

1. Overall health assessment
2. Any degraded or problematic components
3. Recommended actions to resolve issues
4. Priority ordering of issues to address

Health data:
%v

Provide your response in clear markdown format.`, clusterName, healthData)
}

func PolicyCompliancePrompt(clusterName string, policies interface{}) string {
	return fmt.Sprintf(`You are an expert on RHACM (Red Hat Advanced Cluster Management) policy compliance. Analyze the policy compliance status for cluster "%s" and provide:

1. Summary of compliant vs non-compliant policies
2. Impact assessment of non-compliant policies
3. Remediation steps for each non-compliant policy
4. Best practices for maintaining compliance

Policy data:
%v

Provide your response in clear markdown format.`, clusterName, policies)
}

func BMCErrorPrompt(clusterName string, bmcData interface{}) string {
	return fmt.Sprintf(`You are an expert on bare metal server management and Redfish/BMC operations. Analyze the BMC status for cluster "%s" hosts and provide:

1. Current hardware state assessment
2. Any errors or warnings and their meaning
3. Recommended actions
4. Troubleshooting steps for common BMC issues

BMC data:
%v

Provide your response in clear markdown format.`, clusterName, bmcData)
}

func GeneralDiagnosePrompt(context map[string]interface{}) string {
	return fmt.Sprintf(`You are an expert OpenShift ZTP (Zero Touch Provisioning) engineer and cluster administrator. Analyze the following context and provide diagnostic insights:

1. Identify any issues or anomalies
2. Provide root cause analysis
3. Suggest remediation steps
4. List any related issues to watch for

Context:
%v

Provide your response in clear markdown format.`, context)
}
