// Code generated by go generate; DO NOT EDIT.
// 2021-12-06 16:54:35.29554 +0100 CET m=+0.000623709

package ooapi

//go:generate go run ./internal/generator -file swagger_test.go

const swagger = `{
    "swagger": "2.0",
    "info": {
        "title": "OONI API specification",
        "version": "0.20211206.12155435"
    },
    "host": "api.ooni.io",
    "basePath": "/",
    "schemes": [
        "https"
    ],
    "paths": {
        "/api/_/check_report_id": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "query",
                        "name": "report_id",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "error": {
                                    "type": "string"
                                },
                                "found": {
                                    "type": "boolean"
                                },
                                "v": {
                                    "type": "integer"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/check-in": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "body",
                        "name": "body",
                        "required": true,
                        "schema": {
                            "properties": {
                                "charging": {
                                    "type": "boolean"
                                },
                                "on_wifi": {
                                    "type": "boolean"
                                },
                                "platform": {
                                    "type": "string"
                                },
                                "probe_asn": {
                                    "type": "string"
                                },
                                "probe_cc": {
                                    "type": "string"
                                },
                                "run_type": {
                                    "type": "string"
                                },
                                "software_name": {
                                    "type": "string"
                                },
                                "software_version": {
                                    "type": "string"
                                },
                                "web_connectivity": {
                                    "properties": {
                                        "category_codes": {
                                            "items": {
                                                "type": "string"
                                            },
                                            "type": "array"
                                        }
                                    },
                                    "type": "object"
                                }
                            },
                            "type": "object"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "probe_asn": {
                                    "type": "string"
                                },
                                "probe_cc": {
                                    "type": "string"
                                },
                                "tests": {
                                    "properties": {
                                        "web_connectivity": {
                                            "properties": {
                                                "report_id": {
                                                    "type": "string"
                                                },
                                                "urls": {
                                                    "items": {
                                                        "properties": {
                                                            "category_code": {
                                                                "type": "string"
                                                            },
                                                            "country_code": {
                                                                "type": "string"
                                                            },
                                                            "url": {
                                                                "type": "string"
                                                            }
                                                        },
                                                        "type": "object"
                                                    },
                                                    "type": "array"
                                                }
                                            },
                                            "type": "object"
                                        }
                                    },
                                    "type": "object"
                                },
                                "v": {
                                    "type": "integer"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/login": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "body",
                        "name": "body",
                        "required": true,
                        "schema": {
                            "properties": {
                                "password": {
                                    "type": "string"
                                },
                                "username": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "expire": {
                                    "type": "string"
                                },
                                "token": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/measurement_meta": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "query",
                        "name": "report_id",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "in": "query",
                        "name": "full",
                        "type": "boolean"
                    },
                    {
                        "in": "query",
                        "name": "input",
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "anomaly": {
                                    "type": "boolean"
                                },
                                "category_code": {
                                    "type": "string"
                                },
                                "confirmed": {
                                    "type": "boolean"
                                },
                                "failure": {
                                    "type": "boolean"
                                },
                                "input": {
                                    "type": "string"
                                },
                                "measurement_start_time": {
                                    "type": "string"
                                },
                                "probe_asn": {
                                    "type": "integer"
                                },
                                "probe_cc": {
                                    "type": "string"
                                },
                                "raw_measurement": {
                                    "type": "string"
                                },
                                "report_id": {
                                    "type": "string"
                                },
                                "scores": {
                                    "type": "string"
                                },
                                "test_name": {
                                    "type": "string"
                                },
                                "test_start_time": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/register": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "body",
                        "name": "body",
                        "required": true,
                        "schema": {
                            "properties": {
                                "available_bandwidth": {
                                    "type": "string"
                                },
                                "device_token": {
                                    "type": "string"
                                },
                                "language": {
                                    "type": "string"
                                },
                                "network_type": {
                                    "type": "string"
                                },
                                "password": {
                                    "type": "string"
                                },
                                "platform": {
                                    "type": "string"
                                },
                                "probe_asn": {
                                    "type": "string"
                                },
                                "probe_cc": {
                                    "type": "string"
                                },
                                "probe_family": {
                                    "type": "string"
                                },
                                "probe_timezone": {
                                    "type": "string"
                                },
                                "software_name": {
                                    "type": "string"
                                },
                                "software_version": {
                                    "type": "string"
                                },
                                "supported_tests": {
                                    "items": {
                                        "type": "string"
                                    },
                                    "type": "array"
                                }
                            },
                            "type": "object"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "client_id": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/test-helpers": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/test-list/psiphon-config": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/test-list/tor-targets": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/api/v1/test-list/urls": {
            "get": {
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "query",
                        "name": "category_codes",
                        "type": "string"
                    },
                    {
                        "in": "query",
                        "name": "country_code",
                        "type": "string"
                    },
                    {
                        "in": "query",
                        "name": "limit",
                        "type": "integer"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "metadata": {
                                    "properties": {
                                        "count": {
                                            "type": "integer"
                                        }
                                    },
                                    "type": "object"
                                },
                                "results": {
                                    "items": {
                                        "properties": {
                                            "category_code": {
                                                "type": "string"
                                            },
                                            "country_code": {
                                                "type": "string"
                                            },
                                            "url": {
                                                "type": "string"
                                            }
                                        },
                                        "type": "object"
                                    },
                                    "type": "array"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/report": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "body",
                        "name": "body",
                        "required": true,
                        "schema": {
                            "properties": {
                                "data_format_version": {
                                    "type": "string"
                                },
                                "format": {
                                    "type": "string"
                                },
                                "probe_asn": {
                                    "type": "string"
                                },
                                "probe_cc": {
                                    "type": "string"
                                },
                                "software_name": {
                                    "type": "string"
                                },
                                "software_version": {
                                    "type": "string"
                                },
                                "test_name": {
                                    "type": "string"
                                },
                                "test_start_time": {
                                    "type": "string"
                                },
                                "test_version": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "backend_version": {
                                    "type": "string"
                                },
                                "report_id": {
                                    "type": "string"
                                },
                                "supported_formats": {
                                    "items": {
                                        "type": "string"
                                    },
                                    "type": "array"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        },
        "/report/{report_id}": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "parameters": [
                    {
                        "in": "path",
                        "name": "report_id",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "in": "body",
                        "name": "body",
                        "required": true,
                        "schema": {
                            "properties": {
                                "content": {
                                    "type": "object"
                                },
                                "format": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "all good",
                        "schema": {
                            "properties": {
                                "measurement_uid": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    }
                }
            }
        }
    }
}`
