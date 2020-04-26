create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED"; \
alter user {{name}} set PASSWORD = '{{password}}'; \
alter user {{name}} set DEFAULT_ROLE = ROLEFORVAULTROLE; \
grant role ROLEFORVAULTROLE to user {{name}}; \
alter user {{name}} set default_warehouse = "WHFORVAULTROLE"; \
grant usage on warehouse WHFORVAULTROLE to role ROLEFORVAULTROLE; \
alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}}; \


create user {{name}} LOGIN_NAME='{{name}}' FIRST_NAME = "VAULT" LAST_NAME = "CREATED"; alter user {{name}} set PASSWORD = '{{password}}'; alter user {{name}} set DEFAULT_ROLE = vaulttesting; grant role vaulttesting to user {{name}}; alter user {{name}} set default_warehouse = "VAULTTEST"; grant usage on warehouse VAULTTEST to role vaulttesting; alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};