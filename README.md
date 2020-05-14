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
This _sample_ Hashicorp Vault database plugin works with the Snowflake Data Platform. If you are familiar with Vault built-in database plugins, this plugin provides the same features and works the same way. Otherwise, see  Vault's [general concepts](https://www.vaultproject.io/docs/secrets/databases) and [detailed usage](https://www.vaultproject.io/api/secret/databases) documentation to get started. Known limitations are noted below.

## Requirements
This plugin was developed using Go version 1.14.2 on Ubuntu 20.04 LTS. Compatibility with other versions and systems is unknown.

### Environment Requirements
1. **A working Vault install** with the database secrets backend active.
2. **A Go build environment**  and **the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html)**. Both are required to build a binary version of the plugin.

### Snowflake Configuration Requirements    
1. **A Snowflake user with at least the USERADMIN role granted.** Required to create a plugin command or to run a command.
2. **A Snowflake instance configured with the WAREHOUSE and ROLE objects.** Required if using dynamically-created Users-based Vault roles. Dynamic user roles that are owned by the Snowflake user, or granted to the user, use the WAREHOUSE and ROLE objects.
5. **A user owned by the USERADMIN role,** which will be controlled by this plugin .

## Setting Up A Minimalist System
This section describes how to test this plugin in a development setting using a minimalist setup. This ReadMe file does not comment on adapting this SAMPLE for uses other than testing in a dev environment.

### Starting with Hashicorp Vault in Dev mode
After you [install Vault](https://learn.hashicorp.com/vault/getting-started/install), you will [start it in "dev mode"](https://learn.hashicorp.com/vault/getting-started/dev-server). At this stage, you simply want to ensure it works as expected. You can use the [simplest possible secret](https://learn.hashicorp.com/vault/getting-started/first-secret) to test this.

Next prepare the server to run the SAMPLE plugin. Start by [following the procedure for the mock plugin](https://learn.hashicorp.com/vault/developer/plugin-backends). If this works as expected, proceed with this SAMPLE.

### Building the snowflakepasswords-database-plugin
If you followed the mock plugin procedure above, you should have a working golang environment on your system. To build the snowflakepasswords-database-plugin plugin binary, follow these steps:

1. cd to the `vault-snowflakepasswords-sample` in the cloned repo.
2. `go mod init github.com/sanderiam/vault-snowflakepasswords-sample`
3. `go build`
4. `go build snowflakepasswords-database-plugin/main.go`
5. `mv ./main <YOUR VAULT PLUGINS DIRECTORY>/snowflakepasswords-database-plugin`
6. `sha256sum ../plugins/snowflakepasswords-database-plugin | awk '{print $1}'`
7. Save the output of step #6 for a later step, which will be referred to as `<THESHA256>`.

### Enabling the snowflakepasswords-database-plugin in Vault
To enable the plugin with your dev Vault server, follow these steps:

1. Start the Vault server in dev mode and point to `<YOUR VAULT PLUGINS DIRECTORY>` as used above, _e.g._ `vault server -dev -dev-root-token-id=root -dev-plugin-dir=./plugins`
This step takes over that session and you will need a second one to continue.
2. Prepare your session to interact with the running Vault server by setting the `VAULT_ADDR`, _e.g._ `export VAULT_ADDR='http://127.0.0.1:8200'`
3. Log in to Vault, _e.g._ `vault login root`
5. Enable the built-in Vault Database Backend by running `vault secrets enable database`
6. Enable the snowflakepasswords-database-plugin by running `vault write sys/plugins/catalog/database/snowflakepasswords-database-plugin sha256=<THESHA256> command="snowflakepasswords-database-plugin"` - where `<THESHA256>` is the value saved from step #6 of "Building the snowflakepasswords-database-plugin."

Look for this indicator of success: `Success! Data written to: sys/plugins/catalog/database/snowflakepasswords-database-plugin`.

## Using the snowflakepasswords-database-plugin
If the SAMPLE is running successfully, you're  ready to use the plugin features.

### Preparing Snowflake for the snowflakepasswords-database-plugin
If you have not yet created the Snowflake user described in item #1 in the "Snowflake Configuration Requirements" section, create the user now. You need the appropriate Snowflake rights to do this. In this example, the User name is `karnak`. For a full discussion on user creation in Snowflake, see the [CREATE USER topic](https://docs.snowflake.com/en/sql-reference/sql/create-user.html) in the Snowflake product documentation.

To create the user:

```
create user karnak PASSWORD = '<YOURUSERADMINUSERPASSWORD>' DEFAULT_ROLE = USERADMIN;
grant role USERADMIN to user karnak;
```

The `USERADMIN` role provides the minimum rights necessary to accomplish the examples documented in this ReadMe file. Additional rights may be required to have the plugin do things differently. For a full discussion of access control in Snowflake, see the [Access Control in Snowflake](https://docs.snowflake.com/en/user-guide/security-access-control.html) topic in the product documentation. To allow Vault to manage (rotate) the credentials for the `karnak` user, grant ownership of that user to the `USERADMIN` role:

```
grant ownership on user karnak to role USERADMIN;
```

Because users in Snowflake typically need to run SQL and need a role other than the `PUBLIC` role, create the assets and grant the `USERADMIN` role access to manage these so the `karnak` user in Vault can do what it needs to do:

```
CREATE WAREHOUSE VAULTTEST WITH WAREHOUSE_SIZE = 'XSMALL' WAREHOUSE_TYPE = 'STANDARD' AUTO_SUSPEND = 60 AUTO_RESUME = TRUE MIN_CLUSTER_COUNT = 1 MAX_CLUSTER_COUNT = 2 SCALING_POLICY = 'STANDARD';
create role vaulttesting;
grant ownership on role vaulttesting to role USERADMIN;
grant ownership on warehouse VAULTTEST to role USERADMIN;
```

### Connecting Vault to Snowflake
This section requires the Snowflake user we previously named `karnak`. Here we will run a command to set up one of your Snowflake accounts in Vault:

> `vault write database/config/va_demo07 plugin_name=snowflakepasswords-database-plugin allowed_roles="xvi" connection_url="{{username}}:{{password}}@va_demo07.us-east-1/" username="karnak" password="<YOURUSERADMINUSERPASSWORD>"`

If we break down this command, the important pieces are:
* `database/config/va_demo07` - this tells Vault to make a new configuration in the database backend for a Snowflake account it will know as `va_demo07`. This example uses the Snowflake Account's name as the name of the configuration entry, but it's not required that you do that. It can be named whatever you wish.
* `plugin_name=snowflakepasswords-database-plugin` - this references the plugin enabled in the last section.
* `allowed_roles="xvi` - Vault will always put things in the context of roles to authorize access to functions Vault offers. In this example walkthrough, I'm using a net new role I've created named `xvi`, but you can connect this to roles you already have. There is no special role required and it can be used with any roles you wish.
* `connection_url="{{username}}:{{password}}@va_demo07.us-east-1/"` - this is the connection string the plugin uses to call out to Snowflake. This plugin is written in go/golang and uses the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html). The format of this connection string is what is used in this driver.
* `username="karnak" password="<YOURUSERADMINUSERPASSWORD>"` - this is the username and password for the Snowflake user with the USERADMIN privilege (from "Snowflake Configuration Requirements" item #1). These are the initial credentials for this user, and after this you can use this plugin to rotate and manage those credentials from that point on (which will be covered below).

### Rotating The Snowflake Plugin Vault Credentials
Now that Vault is controlling credentials, the first natural thing is to ensure it also has control of its own credentials. This can be accomplished using the following command (or equivalent API call), and can be automated in any way orchestration is convenient for you.

```
vault write -force database/rotate-root/va_demo07
```

If we break down this command, the important pieces are:
* `-force` - this makes the command go through for sure (may not be needed).
* `database/` - reference to being in the database backend again.
* `rotate-root/va_demo07` - the instruction is to rotate the "root" credentials for the `va_demo07` Snowflake Account we connected at the start.

### Setting Up An Ephemeral Snowflake User with Vault

> `vault write database/roles/xvi db_name=va_demo07 creation_statements="create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = \"VAULT\" LAST_NAME = \"CREATED\"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = \"VAULTTEST\"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}" default_ttl=1h max_ttl=2h`

If we break down this command, the important pieces are:
* `database/roles/xvi` - names the role we are creating. Note that the `xvi` role was named when we created the `va_demo07` Snowflake Account definition. If the role was not allowed then, this write would fail because the role would not be authorized. If you want to name this role something else or need to authorize other roles in the future, run `vault write database/config/va_demo07 allowed_roles="xvi,astonishing,mercs"` and write allowed roles as needed.
* `creation_statements="create user {{name}} LOGIN_NAME='{{name}}'...` - creates a Snowflake user every time it's called. To do that it needs instructions for creating a Snowflake user. This allows you to define the SQL used in that process, which means you can use this to alter the user creation process for each distinct role as you need. PLEASE NOTE: These steps recommend using the Snowflake `USERADMIN` role for authorization. If you add SQL that exceeds what the role can do, the operation will fail.

For easier readability all the SQL is listed here:
  * `create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED";` - everything that appears in `{{this}}` format is replaced by the runtime values in the code.
  * `alter user {{name}} set PASSWORD = '{{password}}';`
  * `alter user {{name}} set DEFAULT_ROLE = vaulttesting;` - `DEFAULT_ROLE` will likely vary between different Vault roles you define.
  * `grant role vaulttesting to user {{name}};`
  * `alter user {{name}} set default_warehouse = "VAULTTEST";` - `default_warehouse` will likely vary between different Vault roles you define.
  * `grant usage on warehouse VAULTTEST to role vaulttesting;`
  * `alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};` - Snowflake does not expire users in hours, so this is [calculated as days in the plugin code](https://github.com/sanderiam/vault-snowflakepasswords-sample/blob/f35d2a3b9cc2c356b8b26d12754d9fd12e870bbe/vault-snowflakepasswords-sample.go#L362) and can be set to a single day for every value of hours below 24.
* `default_ttl=1h max_ttl=2h` - sets the default and max lease lifetimes for any user created using this role.

Once you have the role defined, the way to use it is reading from it to generate a user. This would look like this on the command line:

```
$ vault read database/creds/xvi
Key                Value
---                -----
lease_id           database/creds/xvi/Jk8qolr98MgcwFoPo9Kib5xn
lease_duration     1h
lease_renewable    true
password           A1a-randomCHARsWz9z
username           v_token_xvi_hKg9wm9R7Bj98EWxGnXa_1589390049
```
For the next hour (until the lease expires) this user exists with that password, and when the lease expires, the user is dropped by Vault.

### Auditing the Actions of Vault in Snowflake
Now that Vault is running commands and creating users, you may wish to see what it's up to in Snowflake. There are many ways to do this, and for a full discussion of that please see the [Account Usage docs](https://docs.snowflake.com/en/sql-reference/account-usage.html). A quick thing you can do to see what Vault is up to is show all users that it creates:

```
show users like '%token%';
```

Or you can get a complete record of the queries it is running:

```
select QUERY_ID, QUERY_TEXT, USER_NAME, ERROR_CODE, ERROR_MESSAGE, START_TIME
from table(snowflake.information_schema.query_history(dateadd('hours',-4,current_timestamp()),current_timestamp()))
where USER_NAME like '%KAR%' order by start_time DESC;
```

This is looking for a user with a name like `KAR`, but if you used a different root user for Vault you should alter that part of the SQL.

### Using Vault to Rotate Existing Users' Credentials

```
create user bob;
grant OWNERSHIP on user bob to role USERADMIN;
```

> `vault write /database/static-roles/teamdp username="bob" rotation_period="5m" db_name="va_demo07" rotation_statements="alter user {{name}} set password='{{password}}';"`



## Known Limitations

* None at this time.
