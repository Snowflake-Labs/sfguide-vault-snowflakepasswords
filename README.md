# samplehashivaultsnowflakepasswords
snowflakepasswords-database-plugin is a Hashicorp Vault database plugin to manage Snowflake ephemeral users with passwords which may turn into a general password management plugin

## General Usage
If you are already familiar with the [general concepts](https://www.vaultproject.io/docs/secrets/databases) and the [detailed usage](https://www.vaultproject.io/api/secret/databases) of Hashicorp Vault database plugins, then you'll find this is simply a version of that concept which has been adapted to talk to Snowflake's Data Platform. All the features from the built in database plugin have been created here. Known limitations will be noted below. 

## Requirements
1. A working Vault install with the database secrets backend active.
2. A Go build environment to create the binary version of this plugin.
  + This was developed using Go version 1.14.2, and compatability with other versions is unknown.
  + The code uses a number of modules which will need to be present during building, including the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html).
3. A Snowflake user with at least USERADMIN role granted.
4. If you will be using dynamicly created Snowflake Users based vault roles, you will need WAREHOUSE and ROLE objects in Snowflake which will be used by the dynamic users owned or granted to the user in #3 with grant option.
5. Any user which will be controlled by this system should be owned by USERADMIN role.

## Setting Up A Minimally Working System

## Known Limitations
