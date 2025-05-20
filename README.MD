This module can be used to compare two Terraform Plans. This is useful to predict the diff of a change without having to deploy any resources first.

All files named terraform_plan_diff_* were copied from https://github.com/cloudposse/atmos and modified to work outside of the Atmos framework.
Also, instead of just printing the diff to output, it is returned in a map for further processing.