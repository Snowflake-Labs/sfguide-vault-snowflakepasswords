/*
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
*/


-- statements here are for reability
create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED"; \
alter user {{name}} set PASSWORD = '{{password}}'; \
alter user {{name}} set DEFAULT_ROLE = ROLEFORVAULTROLE; \
grant role ROLEFORVAULTROLE to user {{name}}; \
alter user {{name}} set default_warehouse = "WHFORVAULTROLE"; \
grant usage on warehouse WHFORVAULTROLE to role ROLEFORVAULTROLE; \
alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}; \

-- this is how they ought to be fed into the system
create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = "VAULTTEST"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};
