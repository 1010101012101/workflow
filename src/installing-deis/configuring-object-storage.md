# Configure Object Storage

A variety of Deis components rely on an object storage system to do their work. These components are:

- [builder](https://github.com/deis/builder)
- [slugbuilder](https://github.com/deis/slugbuilder)
- [slugrunner](https://github.com/deis/slugrunner)
- [registry](https://github.com/deis/registry)
- [database](https://github.com/deis/postgres)

These components are built flexibly, so they can work out of the box with almost any system that is compatible with the [S3 API](http://docs.aws.amazon.com/AmazonS3/latest/API/APIRest.html).

# Minio

Additionally, Deis ships with a [Minio](http://minio.io) [component](https://github.com/deis/minio). This component runs as a Kubernetes service, and the components listed above are configured to automatically look for that service and use it as object storage if it's available.

# Telling Deis What to Use

The Deis components determine what object storage system to use via environment variables that you set up:

- `DEIS_MINIO_SERVICE_HOST` and `DEIS_MINIO_SERVICE_PORT` - The in-cluster Minio service
- `DEIS_OUTSIDE_STORAGE_HOST` and `DEIS_OUTSIDE_STORAGE_PORT` - The external S3-compatible object storage system

# Storing Credentials

In the Deis V2 Beta release, all components read credentials from the filesystem, and we suggest that credentials are stored in [Kubernetes secrets](http://kubernetes.io/v1.1/docs/user-guide/secrets.html) and mounted to the appropriate location for the component. See the below list for the expected location for each component, and see [the deis-dev chart](https://github.com/deis/charts/tree/master/deis-dev) for examples of using and mounting secrets.

- [builder](https://github.com/deis/builder)
  - Key: `/var/run/secrets/object/store/access-key-id`
  - Secret `/var/run/secrets/object/store/access-key-secret`
- [slugbuilder](https://github.com/deis/slugbuilder)
  - Key: `/var/run/secrets/object/store/access-key-id`
  - Secret `/var/run/secrets/object/store/access-key-secret`
- [slugrunner](https://github.com/deis/slugrunner)
  - Key: `/var/run/secrets/object/store/access-key-id`
  - Secret: `/var/run/secrets/object/store/access-key-secret`
- [registry](https://github.com/deis/registry)
  - Key: `/var/run/secrets/deis/registry/creds/accesskey`
  - Secret: `/var/run/secrets/deis/registry/creds/secretkey`
- [database](https://github.com/deis/postgres)
  - Key: `/etc/wal-e.d/env/access-key-id`
  - Secret: `/etc/wal-e.d/env/access-key-secret`

# A Note on Google Cloud Storage

As you may know Google Cloud Storage (GCS) can [interoperate with the S3 API](https://cloud.google.com/storage/docs/interoperability), and, if you choose to use Google Cloud Storage for object storage, you'll have to turn on this interoperability mode.

If you choose to use Google Cloud Storage, set your `DEIS_OUTSIDE_STORAGE_HOST` environment variable to `storage.googleapis.com`, and follow [these instructions](https://cloud.google.com/storage/docs/migrating?hl=en_US#keys) to generate an S3 compatible access key ID and access key secret. Store these credentials just as you would if they were AWS S3 or Minio credentials (see the "Storing Credentials" section above).

# Limitations

The only currently known limitation is that [the Deis registry component](https://github.com/deis/registry) will not automatically look up the minio service, nor will it look for other storage env vars. That fix is being tracked in a [GitHub issue](https://github.com/deis/registry/issues/7) and is planned for our beta release.
