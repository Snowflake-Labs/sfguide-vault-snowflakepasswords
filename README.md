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
3. A Snowflake user with at least USERADMIN role granted to run the plugin's commands, or the ability to create one.
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

1. Start the vault server in dev mode and point to `<YOUR VAULT PLUGINS DIRECTORY>` as used above, e.g. `vault server -dev -dev-root-token-id=root -dev-plugin-dir=./plugins` 
2. The first step will take over that session, and you will need a second one to continue.
3. Prepare your session to interact with the running vault server by setting the `VAULT_ADDR`, e.g. `export VAULT_ADDR='http://127.0.0.1:8200'`
4. Log in to Vault, e.g. `vault login root`
5. Enable the built in Vault Database Backend by running `vault secrets enable database`
6. Enable the snowflakepasswords-database-plugin by running `vault write sys/plugins/catalog/database/snowflakepasswords-database-plugin sha256=<THESHA256> command="snowflakepasswords-database-plugin"` - where `<THESHA256>` is the value saved from step #6 of "Building the snowflakepasswords-database-plugin". You're looking for this indicator of success: `Success! Data written to: sys/plugins/catalog/database/snowflakepasswords-database-plugin`.

## Using the snowflakepasswords-database-plugin
If you're following along with a dev mode Vault server or using a different set up and you've been able to get this SAMPLE running, you're now ready to use the features. 

### Preparing Snowflake for the snowflakepasswords-database-plugin
This will assume you have not yet created the Snowflake user from the "Requirements" item #3, and will create that user now. You will need the approrpiate Snowflake rights to do this. In this example, we will use a User named `karnak`. For a full discussion on user creation in Snowflake, please see [the CREATE USER section of our docs](https://docs.snowflake.com/en/sql-reference/sql/create-user.html). You can create the user like so:

```
create user karnak PASSWORD = '<YOURUSERADMINUSERPASSWORD>' DEFAULT_ROLE = USERADMIN;
grant role USERADMIN to user karnak;
```

The `USERADMIN` role is the minimum required rights to accomplish the examples used with thie SAMPLE here. If you choose to have the plugin do things differently, more rights may be required. For a full discussion of access dontrol in Snowlfake, please see [Access Control in Snowflake in our docs](https://docs.snowflake.com/en/user-guide/security-access-control.html). If you wish to allow Valut to manage (rotate) the credentials for the `karnak` user as well, you will need to grant ownership of that user to the `USERADMIN` role - like so:

```
grant ownership on user karnak to role USERADMIN;
```

Since users in SNowflake will likley need to run SQL and will also likely need a role other than the `PUBLIC` role, you will want to create these assets and grant the `USERADMIN` role access to manage these so the `karnak` user Vault will use can do what it needs to do. That can be done liek so:

```
CREATE WAREHOUSE VAULTTEST WITH WAREHOUSE_SIZE = 'XSMALL' WAREHOUSE_TYPE = 'STANDARD' AUTO_SUSPEND = 60 AUTO_RESUME = TRUE MIN_CLUSTER_COUNT = 1 MAX_CLUSTER_COUNT = 2 SCALING_POLICY = 'STANDARD';
create role vaulttesting;
grant ownership on role vaulttesting to role USERADMIN;
grant ownership on warehouse VAULTTEST to role USERADMIN;
```

### Connecting Vault to Snowflake
In order to get started, you will need the Snowflake user from the "Requirements" item #3. In this example, we will use a User named `karnak`. Assuming you're continuing from the last section (or that you know what you're doing well enough), the next step is to run a command to set up one of your Snowflake Accounts in Vault. This setup command will look something like this:

> `vault write database/config/va_demo07 plugin_name=snowflakepasswords-database-plugin allowed_roles="xvi" connection_url="{{username}}:{{password}}@va_demo07.us-east-1/" username="karnak" password="<YOURUSERADMINUSERPASSWORD>"`

If we break down this command, the important pieces are:
* `database/config/va_demo07` - this tells vault to make a new configuration in the datbase backend for a Snowflake account it will know as `va_demo07`. In this example, I've used the Snowflake Account's name as the name of the configuration entry, but it's not required that you do that. It can be named whatever you wish. 
* `plugin_name=snowflakepasswords-database-plugin` - this references the plugin enabled in the last section.
* `allowed_roles="xvi` - Vault will always put things in the context of roles to authorize access to functions Vault offers. In this exmaple walkthrough, I'm using a net new role I've created named `xvi`, but it's likely  you may connect this to roles you already have. There is no special role required and it can be used with any roles you wish. 
* `connection_url="{{username}}:{{password}}@va_demo07.us-east-1/"` - this is the connection string this plugin will use to call out to Snowflake. This plugin is written in go/golang and uses the [Snowflake Go Driver](https://docs.snowflake.com/en/user-guide/go-driver.html). The format of this connection string is what is used in this driver. 
* `username="karnak" password="<YOURUSERADMINUSERPASSWORD>"` - this is the username and password for the Snowflake user with USERADMIN privilege (from the "Requirements" item #3). These are the initial credentials for this user, and after this you can use this plugin to rotate and manage those credentials from that point on (which will be covered below).

### Rotating The Snowflake Plugin Vault Crednetials
Now that Vault is controling credentials, the first natural thing is to ensure it also has control of its own credentials. This can be accomplished using the following command (or equivalent API call), and can be ausotmated in any way orchestration is convienent for you. 

```
vault write -force database/rotate-root/va_demo07
```

If we break down this command, the important pieces are:
* `-force` - this makes the command go through for sure (may not be needed).
* `database/` - reference to being in the database backend again.
* `rotate-root/va_demo07` - the instruction is to rotate the "root" credentials for the `va_demo07` Snowflake Account we conected at the start.

### Setting Up An Ephemeral Snowflake User with Vault

> `vault write database/roles/xvi db_name=va_demo07 creation_statements="create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = \"VAULT\" LAST_NAME = \"CREATED\"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = \"VAULTTEST\"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}" default_ttl=1h max_ttl=2h`

If we break down this command, the important pieces are:
* `database/roles/xvi` - this is naming the role we are creating. It's important to note that this role `xvi` was named when we created the `va_demo07` Snowflake Account definition. If the role were not allowed then, this write woudl fail as the role would not be authorized. If you wanted to name this role something else or needed to authorize other roles in the future you can run `vault write database/config/va_demo07 allowed_roles="xvi,astonishing,mercs"` and write allowed roles as needed. 
* `creation_statements="create user {{name}} LOGIN_NAME='{{name}}'...` - thie role will create a SNowflake user every time it's called. In order to do that it needs to have the instructions for how a Snowflake user is created. This allows you to give all the SQL used in that process. That means you can use this to alter the user creation process for each distinct role as you need. PLEASE NOTE, this set of instructions assumes everything will be done in Snowflake using hte `USERADMIN` role as authorization and does not attempt to do anything the role is not authorized to do. If you put in SQL to these defintions which falls outside what that role can do, it will fail. For easier readability all the SQL used is listed here:
  * `create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED";` - everything that appears in `{{this}}` format will be replaced by the run time values in the code.
  * `alter user {{name}} set PASSWORD = '{{password}}';`
  * `alter user {{name}} set DEFAULT_ROLE = vaulttesting;` - `DEFAULT_ROLE` is something likely to vary between diffeent Vault roles you define.
  * `grant role vaulttesting to user {{name}};`
  * `alter user {{name}} set default_warehouse = "VAULTTEST";` - `default_warehouse` is something likely to vary between diffeent Vault roles you define.
  * `grant usage on warehouse VAULTTEST to role vaulttesting;`
  * `alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};` - Snowflake does not expire users in hours, so this will be [calculated as days in the plugin code](https://github.com/sanderiam/vault-snowflakepasswords-sample/blob/f35d2a3b9cc2c356b8b26d12754d9fd12e870bbe/vault-snowflakepasswords-sample.go#L362) and be set to a single day for every value of hours below 24.
* `default_ttl=1h max_ttl=2h` - this sets the default and max lease lifetimes for any user created using this role. 

Once you have the role defined, the way to use it is reading from it to generate a user. This would look liek this on the command line:

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
For the next hour (until the lease expires) this user would exist with that password. At the end that user will be dropped by Vault. 

### Auditing the Actions of Vault in Snowflake
Now that Vault is running commands and creating users, you may wish to see what it's up to in Snowflake. There are many ways to do this, and for a full discussion of that please see [our Account Usage docs](https://docs.snowflake.com/en/sql-reference/account-usage.html). Two quick things you can do to see what Vault is up to is show all users that it creates:

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

### Using Vault to Rotate Existing Users' Crednetials

```
create user bob;
grant OWNERSHIP on user bob to role USERADMIN;
```

> `vault write /database/static-roles/teamdp username="bob" rotation_period="5m" db_name="va_demo07" rotation_statements="alter user {{name}} set password='{{password}}';"`



## Known Limitations
