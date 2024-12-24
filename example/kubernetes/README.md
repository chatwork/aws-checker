# aws-checker Kubernetes and Datadog example

This directory contains a set of files and scripts to deploy `aws-checker` onto AWS EKS (Auto Mode preferred), with the metrics collected and available in Datadog.

## Contents

- `aws-checker`: The file uploaded to S3 for `aws-checker` S3 checks
- `aws-checker.yaml`: A Kubernetes manifest file containing a few static resources (DatadogAgent, ServiceAccount, Deplyoment)
- `secret.sh`: The script to generate `Secret` resource manifest YAML
- `configmap.sh`: The script to generate `ConfigMap` resource manifest YAML
- `manifests.sh`: The script to generate and write all the resource manifest YAML to stdout, piped to `kubectl create -f` and `kubectl replace -f -`

## Prerequisites

- `direnv`
- `kubectl`
- `helm`
- [Datadog Operator](https://docs.datadoghq.com/getting_started/containers/datadog_operator/)

## Usage

1. Create `.env.config` with the following contents:

```shell
S3_BUCKET=<S3 BUCKET NAME>
S3_KEY=<S3 OBJECT KEY>
DYNAMODB_TABLE=<DYNAMODB TABLE NAME>
SQS_QUEUE_URL=https://sqs.<AWS REGION>.amazonaws.com/<AWS ACCOUNT ID>/<QUEUE NAME>
CLUSTER_NAME=<EKS CLUSTER NAME>
```

2. Create AWS resources

You need the following AWS resources in your AWS account:

- A S3 bucket named `<S3 BUCKET NAME>`
  - `aws s3 cp aws-checker s3://<S3 BUCKET NAME>/<S3 KEY>` to upload the object to pass the aws-checker S3 checks
- A DynamoDB table named `<DYNAMODB TABLE NAME>`
- A SQS queue named `<QUEUE NAME>`
- An EKS cluster named `<EKS CLUSTER NAME>`

3. Create `.env.secret` with the following contents:

```shell
AWS_REGION=<AWS REGION>
```

4. Create `.env.local` with the following contents:

```shell
export DD_API_KEY=<DATADOG API KEY>
```

5. Generate and create the resources:

```shell
direnv allow

./manifests.sh | kubectl create -f -
```

6. Verify everything is working

```shell
$ kubectl get po
NAME                                     READY   STATUS    RESTARTS   AGE
aws-checker-5c79ff5f98-q9jvz             1/1     Running   0          17m
datadog-agent-jzg7r                      3/3     Running   0          23m
datadog-cluster-agent-78d79c5c55-t6xx5   1/1     Running   0          23m
my-datadog-operator-7f56c485d9-zdvqw     1/1     Running   0          26m
```

7. Browse metrics

Go to https://app.datadoghq.com/metric/explorer and select `aws_checker_example.aws_request_duration_seconds.count` metrics.

Setting `sum by` to `method`, `status`, and `service` would be a good idea.
