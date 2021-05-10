# PLEASE NOTE: Hashicorp has released an official plugin, which you can find here: https://github.com/hashicorp/vault-plugin-database-snowflake

# Vault database plugin for Snowflake  

> A _sample_ Hashicorp Vault database plugin designed to work with the Snowflake Data Platform.

Copyright © 2020 Snowflake Inc. All rights reserved.

## Contents
* [General usage](#general-usage)
* [Requirements](#requirements)
* [Setting up a minimalist system](#setting-up-a-minimalist-system)
* [Using the plugin](#using-the-plugin)
* [Known limitations](#known-limitations)
* [Apache license](#apache-license)



## General Usage

This _sample_ Hashicorp Vault database plugin works with the Snowflake Data Platform. If you are familiar with Vault built-in database plugins, this plugin provides the same features and works the same way. Otherwise, see  Vault's [general concepts](https://www.vaultproject.io/docs/secrets/databases) and [detailed usage](https://www.vaultproject.io/api/secret/databases) documentation to get started. Known limitations are [noted below](#known-limitations).


## Requirements

The `snowflakepasswords-database-plugin` requires:

1. **A working Vault install** with the database secrets backend active.
2. **A Go build environment**  and the **[Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html)**. Both are required to build a binary version of the plugin.

### Required Snowflake Setup

1. **A Snowflake user with _at least_ the USERADMIN role granted**. Required to create a plugin command or to run a command. See [Preparing Snowflake to work with the plugin](#preparing-snowflake-to-work-with-the-plugin) for steps to create the user.
2. **A Snowflake instance configured with a WAREHOUSE and ROLE for use with dynamically created users**. To run any SQL, dynamically-created Users based on Vault roles will need a warehouse and a role.
3. **A user owned by the USERADMIN role** that will be controlled by this plugin.



## Setting up a Minimalist System
This section describes how to test this plugin in a development setting using a minimalist setup. This README file does not comment on adapting this SAMPLE for uses other than testing in a dev environment.

### Starting with Hashicorp Vault in Dev mode
After you [install Vault](https://learn.hashicorp.com/vault/getting-started/install), you will [start it in "dev mode"](https://learn.hashicorp.com/vault/getting-started/dev-server). At this stage, you simply want to ensure it works as expected. Use the [simplest possible secret](https://learn.hashicorp.com/vault/getting-started/first-secret) to test this.

Next, prepare the server to run the SAMPLE plugin. Start by [following the procedure for the mock plugin](https://learn.hashicorp.com/vault/developer/plugin-backends). If this works as expected, proceed with this SAMPLE.

### Building the Plugin
If you followed the mock plugin procedure above, you should have a working golang environment on your system. To build the `snowflakepasswords-database-plugin` plugin binary, follow these steps:

1. cd to the root directory of the cloned repo, `sfguide-vault-snowflakepasswords` by default.
2. `go mod init github.com/sanderiam/vault-snowflakepasswords-sample` (depending on your go version, you may also need to run `go mod tidy` or equiv at this time)
3. `go build`
4. `go build snowflakepasswords-database-plugin/main.go`
5. `mv ./main <YOUR VAULT PLUGINS DIRECTORY>/snowflakepasswords-database-plugin`
6. `sha256sum <YOUR VAULT PLUGINS DIRECTORY>/snowflakepasswords-database-plugin | awk '{print $1}'`
7. Save the output of step #6 for a later step, which will be referred to as `<THESHA256>`.

### Enabling the Plugin in Vault
To enable the plugin (`snowflakepasswords-database-plugin`) with your dev Vault server, follow these steps:

1. Start the Vault server in dev mode and point to `<YOUR VAULT PLUGINS DIRECTORY>` as used above, _e.g._ `vault server -dev -dev-root-token-id=root -dev-plugin-dir=./plugins`
This step takes over that session and you will need a second one to continue.
2. Prepare your session to interact with the running Vault server by setting the `VAULT_ADDR`, _e.g._ `export VAULT_ADDR='http://127.0.0.1:8200'`
3. Log in to Vault, _e.g._ `vault login root`
5. Enable the built-in Vault Database Backend by running `vault secrets enable database`
6. Enable the snowflakepasswords-database-plugin by running `vault write sys/plugins/catalog/database/snowflakepasswords-database-plugin sha256=<THESHA256> command="snowflakepasswords-database-plugin"` - where `<THESHA256>` is the value saved from step #6 of [Building the snowflakepasswords-database-plugin](#building-the-snowflakepasswords-database-plugin)."

Look for this indicator of success: `Success! Data written to: sys/plugins/catalog/database/snowflakepasswords-database-plugin`.


## Using the Plugin
If the SAMPLE is running successfully, you're  ready to use the plugin features.

### Preparing Snowflake to Work with the Plugin
If you have not yet created the Snowflake user described in the [Required Snowflake Setup](#required-snowflake-setup) section, create the user now. You need the appropriate Snowflake rights to do this. In this example, the User name is `karnak`. For a full discussion on user creation in Snowflake, see the [CREATE USER topic](https://docs.snowflake.com/en/sql-reference/sql/create-user.html) in the Snowflake product documentation.

To create the user:

```
create user karnak PASSWORD = '<YOURUSERADMINUSERPASSWORD>' DEFAULT_ROLE = USERADMIN;
grant role USERADMIN to user karnak;
```

The `USERADMIN` role provides the minimum rights necessary to accomplish the examples documented in this README file. Additional rights may be required to have the plugin do things differently. For a full discussion of access control in Snowflake, see the [Access Control in Snowflake](https://docs.snowflake.com/en/user-guide/security-access-control.html) topic in the product documentation. To allow Vault to manage (rotate) the credentials for the `karnak` user, grant ownership of that user to the `USERADMIN` role:

```
grant ownership on user karnak to role USERADMIN;
```

Because users in Snowflake typically need to run SQL and need a role other than the `PUBLIC` role, create the assets and grant the `USERADMIN` role access to manage these so the `karnak` user in Vault can do what it needs to do:

```
CREATE WAREHOUSE VAULTTEST WITH WAREHOUSE_SIZE = 'XSMALL' WAREHOUSE_TYPE = 'STANDARD' AUTO_SUSPEND = 60 AUTO_RESUME = TRUE MIN_CLUSTER_COUNT = 1
MAX_CLUSTER_COUNT = 2 SCALING_POLICY = 'STANDARD';
create role vaulttesting;
grant ownership on role vaulttesting to role USERADMIN;
grant ownership on warehouse VAULTTEST to role USERADMIN;
```

In this SAMPLE we only use one Snowflake role and warehouse. In a real world scenario, you would have a wide variety of Snowflake roles and warehouses that would be managed by a wide variety of Vault roles, all with different levels of rights and permissions.

### Connecting Vault to Snowflake
This section requires the Snowflake user named `karnak` that we created in the previous section. Running the following command sets up one of your Snowflake accounts in Vault:

> `vault write database/config/uncannyxmen plugin_name=snowflakepasswords-database-plugin allowed_roles="xvi" connection_url="{{username}}:{{password}}@uncannyxmen.us-east-1/" username="karnak" password="<YOURUSERADMINUSERPASSWORD>"`

If we break down this command, the important pieces are:
* `database/config/uncannyxmen` – This tells Vault to make a new configuration in the database backend for a Snowflake account it will know as `uncannyxmen`. This example uses the Snowflake Account's name as the name of the configuration entry, but it's not required that you do that. It can be named whatever you wish.
* `plugin_name=snowflakepasswords-database-plugin` – This references the plugin enabled in the last section.
* `allowed_roles="xvi` - Vault will always put things in the context of roles to authorize access to functions Vault offers. In this example walkthrough, I'm using a net new role I've created named `xvi`, but you can connect this to roles you already have. There is no special role required and it can be used with any roles you wish.
* `connection_url="{{username}}:{{password}}@uncannyxmen.us-east-1/"` – This is the connection string the plugin uses to call out to Snowflake. This plugin is written in go/golang and uses the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html). The format of this connection string is what is used in this driver.
* `username="karnak" password="<YOURUSERADMINUSERPASSWORD>"` – This is the username and password for the Snowflake user with the USERADMIN privilege (from "Snowflake Configuration Requirements" item #1). These are the initial credentials for this user, and after this you can use this plugin to rotate and manage those credentials from that point on (which will be covered below).

### Rotating the Snowflake Plugin Vault Credentials
Now that Vault is controlling credentials, the first natural thing is to ensure it also has control of its own credentials. This can be accomplished using the following command (or the equivalent API call), and can be automated in any way orchestration is convenient for you.

```
vault write -force database/rotate-root/uncannyxmen
```

If we break down this command, the important pieces are:
* `-force` – this makes the command go through for sure (may not be needed).
* `database/` – reference to being in the database backend again.
* `rotate-root/uncannyxmen` – the instruction is to rotate the "root" credentials for the `uncannyxmen` Snowflake Account we connected at the start.

### Setting up an Ephemeral Snowflake User with Vault

> `vault write database/roles/xvi db_name=uncannyxmen creation_statements="create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = \"VAULT\" LAST_NAME = \"CREATED\"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = \"VAULTTEST\"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}" default_ttl=1h max_ttl=2h`

If we break down this command, the important pieces are:
* `database/roles/xvi` – Names the role we are creating. Note that the `xvi` role was named when we created the `uncannyxmen` Snowflake Account definition. If the role was not allowed then, this write would fail because the role would not be authorized. If you want to name this role something else or need to authorize other roles in the future, run `vault write database/config/uncannyxmen allowed_roles="xvi,astonishing,mercs"` and write allowed roles as needed.   

* `creation_statements="create user {{name}} LOGIN_NAME='{{name}}'...` – Creates a Snowflake user every time it is called. To do that it needs instructions for creating a Snowflake user. This lets you define the SQL used in that process, which means you can alter the user creation process for each distinct role as needed.   
  > _PLEASE NOTE:_ These steps recommend using the Snowflake `USERADMIN` role for authorization. If you add SQL that exceeds what the role can do, the operation will fail. All the SQL commands used here are listed below.

* `default_ttl=1h max_ttl=2h` – Sets the default and max lease lifetimes for any user created using this role.

For easier readability all the SQL is listed here:
  * `create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED";` – Everything that appears in `{{this}}` format is replaced by the runtime values in the code.
  * `alter user {{name}} set PASSWORD = '{{password}}';`
  * `alter user {{name}} set DEFAULT_ROLE = vaulttesting;` – The `DEFAULT_ROLE` will likely vary between different Vault roles you define.
  * `grant role vaulttesting to user {{name}};`
  * `alter user {{name}} set default_warehouse = "VAULTTEST";` – The `default_warehouse` will likely vary between different Vault roles you define.
  * `grant usage on warehouse VAULTTEST to role vaulttesting;`
  * `alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};` – Snowflake does not expire users in hours, so this is [calculated as days in the plugin code](https://github.com/sanderiam/vault-snowflakepasswords-sample/blob/f35d2a3b9cc2c356b8b26d12754d9fd12e870bbe/vault-snowflakepasswords-sample.go#L362) and can be set to a single day for every value of hours below 24.


Once you have the role defined, read from it to generate a user. On the command line the command looks like this:

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
For the next hour (until the lease expires) this user exists with that password. When the lease expires, the user is dropped by Vault.

### Auditing the Actions of Vault in Snowflake
Now that Vault is running commands and creating users, you can see what it's up to in Snowflake. There are many ways to do this, and for a full discussion of that please see the [Account Usage docs](https://docs.snowflake.com/en/sql-reference/account-usage.html). A quick thing you can do to see what Vault is up to is show all users that it creates:

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
Along with creating dynamic users, the other common Vault pattern is to manage the passwords of existing Users. This is often applied to User leveraged as service accounts by orchestrated and programmatic tasks.

To test this using this SAMPLE, first create a user to manage and grant ownership of that use to the USERADMIN role.

```
create user bob;
grant OWNERSHIP on user bob to role USERADMIN;
```

The USERADMIN role must own the user, or have rights to manage the user in order for this to work.

Because this extends the Vault configuration for our Snowflake account to a new role, that role must become an `allowed_role` in the configuration. We set that up like so:

```
vault write database/config/uncannyxmen allowed_roles="xvi,astonishing,teamdp"
```

This adds `astonishing` and `teamdp` to the roles that are allowed for `uncannyxmen`. It is the same command used to originally create the `uncannyxmen` configuration, but only writing the single attribute for `allowed_roles`. Note that you must also include the original `xvi` role or it will no longer be allowed.

Next we configure Vault to manage the `bob` user we created.

> `vault write /database/static-roles/teamdp username="bob" rotation_period="5m" db_name="uncannyxmen" rotation_statements="alter user {{name}} set password='{{password}}';"`

If we break down this command, the important pieces are:
* `write /database` - Writes a new configuration to the database backend.
* `/static-roles/teamdp` - Like other constructs, Vault manages this as a role. The type of role is the "static-role". This is understood in comparison to the dynamic roles used in the last example of Vault role creation where a new credential (a new Snowflake User) was created each time the role was called. This time there is a static User and only the user's password is changed. The role is named `teamdp` in this example.
* `username="bob"` -  The User name for this static role.
* `rotation_period="5m"` - This sets how often Vault will change this User's password. It can be measured in increments as small as minutes, but can also be set to hours or days.
* `db_name="uncannyxmen"` - The database name that will hold this role's configuration.
* `rotation_statements="alter user {{name}} set password='{{password}}';"` - This is the SQL that will be run each time Vault reaches the `rotation_period` and executes the configured commands. This example uses the absolute minimum SQL needed to accomplish the task of rotating the credential, but you could extend this to run any SQL needed. The only limitation is that the user running these commands has the rights to do so.

Once you have the role defined, use it by reading from it to change the user's password. For example, on the command line:

```
$ vault read /database/static-creds/teamdp
Key                    Value
---                    -----
last_vault_rotation    2020-05-16T09:32:24.46958155-04:00
password               A1a-QXnOpEnOpEX4G0oc
rotation_period        5m
ttl                    4m52s
username               bob
```

For five minutes this user exists with the specified password, and when the counter expires, Vault changes the user's password. The `ttl` is the countdown to the next time Vault will automatically change the value of the User's password. If you check again in a few moments, you will see that the countdown has diminished:

```
$ vault read /database/static-creds/teamdp
Key                    Value
---                    -----
last_vault_rotation    2020-05-16T09:32:24.46958155-04:00
password               A1a-QXnOpEnOpEX4G0oc
rotation_period        5m
ttl                    3m46s
username               bob
```

You can also force the rotation to happen, which will also reset the `ttl`.

```
$ vault write -force /database/rotate-role/teamdp
Success! Data written to: database/rotate-role/teamdp
$ vault read /database/static-creds/teamdp
Key                    Value
---                    -----
last_vault_rotation    2020-05-16T09:33:56.530157317-04:00
password               A1a-XMnAdAnAdAsNWbK2
rotation_period        5m
ttl                    4m57s
username               bob
```


## Known Limitations

* Most similar Vault database plugins will check for the user's existence before dropping the user. Since that sort of operation requires a warehouse to run SQL and rights the USERADMIN role would not normally have, we've skipped that check. There should be no harm in that, but it is a deviation from the normal pattern.
* This plugin was developed using Go version 1.14.2 on Ubuntu 20.04 LTS. Compatibility with other versions and systems is unknown.



## Apache License

Licensed under the Apache License, Version 2.0 (the  "License"); you may not use this file except in compliance with the License.  
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the License for the specific language governing permissions and limitations under the License.



<br/><br/><br/><br/><br/><br/>
