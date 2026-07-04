# Vendor Dashboard Demo

This example shows a domain-specific vendor/lead analytics dashboard handler.

It is **not** mounted by default in the framework. The SQL expects `vendor`, `lead`, and `campaign` tables that are not part of the core migrations.

To use it in your app, copy the handler into your project and register it behind JWT middleware.
