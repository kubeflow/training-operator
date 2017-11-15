Train a simple TF program to verify we can execute ops.

The program does a simple matrix multiplication.

Only the master assigns ops to devices/workers.

The master will assign ops to every task in the cluster. This way we can verify
that distributed training is working by executing ops on all devices.
