# vault-snowflakepasswords-sample
vault-snowflakepasswords-sample is a SAMPLE Hashicorp Vault database plugin designed to work with the Snowflake Data Platform. 

Copyright (c) 2019 Snowflake Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

## General Usage
If you are already familiar with the [general concepts](https://www.vaultproject.io/docs/secrets/databases) and the [detailed usage](https://www.vaultproject.io/api/secret/databases) of Hashicorp Vault database plugins, then you'll find this is simply a version of that concept which has been adapted to talk to Snowflake's Data Platform. All the features from the built in database plugin have been created here. Known limitations will be noted below. 

## Requirements
1. A working Vault install with the database secrets backend active.
2. A Go build environment to create the binary version of this plugin.
  + This was developed using Go version 1.14.2 on Ubuntu 20.04 LTS, and compatability with other versions and systems is unknown.
  + The code uses a number of modules which will need to be present during building, including the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html).
3. A Snowflake user with at least USERADMIN role granted to run the plugin's commands.
4. If you will be using dynamicly created Snowflake Users based vault roles, you will need WAREHOUSE and ROLE objects in Snowflake which will be used by the dynamic users owned or granted to the user in #3 with grant option.
5. Any user which will be controlled by this plugin must be owned by USERADMIN role.

## Setting Up A Minimally Working System
The assumption in this section is that you wish to test this plugin in a development setting and are looking for the minimal set up to do so. This will not attempt to comment on how to adapt this SAMPLE to use beyond that sort of testing. 

### Starting with Hashicorp Vault in Dev mode
Once you go through the process to [install Vault](https://learn.hashicorp.com/vault/getting-started/install), you will [start it in "dev mode"](https://learn.hashicorp.com/vault/getting-started/dev-server). At this stage, you simply want to ensure it's working as expected. You can use the [simplest possible secret](https://learn.hashicorp.com/vault/getting-started/first-secret) to test this. 

Next you will need to get this server ready for using the SAMPLE plugin. It's best to [follow the procedure for the mock plugin](https://learn.hashicorp.com/vault/developer/plugin-backends) as a starting point. If this works as expected, then you are ready to proceed with this SAMPLE. 

### Building the snowflakepasswords-database-plugin
If you followed the mock plugin procedure above, you should already have a working golang envonment on your system. To build the snowflakepasswords-database-plugin plugin binary, follow these steps:

1. cd to the vault-snowflakepasswords-sample from the cloned repo.
2. `go mod init github.com/sanderiam/vault-snowflakepasswords-sample`
3. `go build`
4. `go build snowflakepasswords-database-plugin/main.go`
5. `mv ./main <YOUR VAULT PLUGINS DIRECTORY>/snowflakepasswords-database-plugin`
6. `sha256sum ../plugins/snowflakepasswords-database-plugin | awk '{print $1}'`
7. Save the output of step #6 for a later step, which will be referred to as `<THESHA256>`.

### Enabling the snowflakepasswords-database-plugin in vault
This again assumes you are using the dev server to understand this SAMPLE. To enable the plugin with your dev Vault server, follow these steps:

1. Start the vault server in dev mode and point to <YOUR VAULT PLUGINS DIRECTORY> as used above, e.g. `vault server -dev -dev-root-token-id=root -dev-plugin-dir=./plugins` 
2. The first step will take over that session, and you will need a second one to continue.
3. Prepare your session to interact with the running vault server by setting the `VAULT_ADDR`, e.g. `export VAULT_ADDR='http://127.0.0.1:8200'`
4. Log in to Vault, e.g. `vault login root`
5. Enable the built in Vault Database Backend by running `vault secrets enable database`
6. Enable the snowflakepasswords-database-plugin by running `vault write sys/plugins/catalog/database/snowflakepasswords-database-plugin sha256=<THESHA256> command="snowflakepasswords-database-plugin"` - where `<THESHA256>` is the value saved from step #6 of "Building the snowflakepasswords-database-plugin". You're looking for this indicator of success: `Success! Data written to: sys/plugins/catalog/database/snowflakepasswords-database-plugin`.

## Using the snowflakepasswords-database-plugin
If you're following along with a dev mode Vault server or using a different set up and you've been able to get this SAMPLE running, you're now ready to use the features. 

### Connecting Vault to Snowflake
In order to get started, you will need the Snowflake user from the "Requirements" item #3. In this example, we will use a User named `karnak`. Assuming you're continuing from the last section (or that you know what you're doing well enough), the next step is to run a command to set up one of your Snowflake Accounts in Vault. This setup command will look something like this:

> `vault write database/config/va_demo07 plugin_name=snowflakepasswords-database-plugin allowed_roles="xvi" connection_url="{{username}}:{{password}}@va_demo07.us-east-1/" username="karnak" password="<YOURUSERADMINUSERPASSWORD>"`

If we break down this command, the important pieces are:
* `database/config/va_demo07` - this tells vault to make a new configuration in the datbase backend for a Snowflake account it will know as `va_demo07`. In this example, I've used the Snowflake Account's name as the name of the configuration entry, but it's not required that you do that. It can be named whatever you wish. 
* `plugin_name=snowflakepasswords-database-plugin` - this references the plugin enabled in the last section.
* `allowed_roles="xvi` - Vault will always put things in the context of roles to authorize access to functions Vault offers. In this exmaple walkthrough, I'm using a net new role I've created, but it's likely in a system you have you may connect this to roles you already have. There is no special role required and it can be used with any roles you wish. 
* `connection_url="{{username}}:{{password}}@va_demo07.us-east-1/"` - this is the connection string this plugin will use to call out to Snowflake. This plugin is written in go/golang and uses the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html). The format of this connection string is what is used in this driver. 
* `username="karnak" password="<YOURUSERADMINUSERPASSWORD>"` - this is the username and password for the Snowflake user with USERADMIN privilege (from the "Requirements" item #3). These are the initial credentials for this user, and after this you can use this plugin to rotate and manage those credentials from that point on (which will be covered below).

### Rotating The Snowflake Plugin Vault Crednetials

### Setting Up An Ephemeral Snowflake User with Vault

> `vault write database/roles/xvi db_name=va_demo07 creation_statements="create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = \"VAULT\" LAST_NAME = \"CREATED\"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = \"VAULTTEST\"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}" default_ttl=1h max_ttl=2h`

```
$ vault read database/creds/xvi
Key                Value
---                -----
lease_id           database/creds/xvi/Jk8qolr98MgcwFoPo9Kib5xn
lease_duration     1h
lease_renewable    true
password           A1a-chwmvxhxt54WzO9z
username           v_token_xvi_hKg9wm9R7Bj98EWxGnXa_1589390049
```

## Known Limitations
