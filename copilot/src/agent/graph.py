"""Minimal, canonical React agent implementation using LangGraph.

This implementation follows the latest LangGraph best practices for creating
a React (Reasoning and Acting) agent with tool integration. State and memory
management is handled automatically by the LangGraph server/platform.
"""

from __future__ import annotations

import os
from typing import Any, Dict

import requests
from dotenv import load_dotenv
from langchain_anthropic import ChatAnthropic
from langchain_community.agent_toolkits.openapi import planner
from langchain_community.agent_toolkits.openapi.spec import reduce_openapi_spec
from langchain_community.utilities.requests import RequestsWrapper
from langchain_core.runnables import RunnableConfig
from langchain_core.tools import tool
from langgraph.prebuilt import create_react_agent
from llama_index.core import Settings
from llama_index.indices.managed.llama_cloud import LlamaCloudIndex

Settings.llm = None

load_dotenv()
BASE_URL = os.environ.get("API_BASE_URL", "https://app.bidon.com").rstrip("/")
SPEC_URL = f"{BASE_URL}/api/openapi.json"
API_BASE = f"{BASE_URL}/api"


def fetch_spec() -> dict:
    r = requests.get(SPEC_URL, timeout=5)
    r.raise_for_status()
    return r.json()


def filter_get_only(spec: dict) -> dict:
    for path, ops in list(spec.get("paths", {}).items()):
        spec["paths"][path] = {m: d for m, d in ops.items() if m.lower() == "get"}
        if not spec["paths"][path]:
            del spec["paths"][path]
    return spec


def reduce_spec(filtered: dict) -> dict:
    return reduce_openapi_spec(filtered)


# write top message description of thee tool purpose
@tool
def query_admin_api(task: str, config: RunnableConfig | None = None) -> str:
    """Query the Admin API via OpenAPI planner (GET-only).
    Fetch Inventory data from the Admin API.
    Task is a natural language instruction to fetch data from the Admin API.
    Internal planner will convert it into a structured request.
    <adminApiContext>
        <hints>
            <hint>Prefer /api/*_collection endpoints for filtered/paginated lists over simple plural nouns.</hint>
            <hint>Always use AuctionConfiguration V2 (/api/v2/auction_configurations*); V1 is deprecated.</hint>
        </hints>

        <entities>
            <entity name="User" desc="Account identity.">
            <fields>id, email</fields>
            <relationships>hasMany: App, DemandSourceAccount, ApiKey;</relationships>
            </entity>

            <entity name="ApiKey" desc="Access key for API.">
            <fields>id(uuid), value(only on create), last_accessed_at</fields>
            <relationships>belongsTo: User (implicit ownership)</relationships>
            </entity>

            <entity name="App" desc="Mobile app under monetization.">
            <fields>id, platform_id(ios|android), human_name, package_name, user_id, app_key, categories[], store_id, store_url, public_uid</fields>
            <relationships>belongsTo: User; hasMany: LineItem, Segment, AuctionConfigurationV2, AppDemandProfile</relationships>
            </entity>

            <entity name="DemandSource" desc="Ad network integration.">
            <fields>id, human_name, api_key, public_uid</fields>
            <relationships>hasMany: DemandSourceAccount, AppDemandProfile</relationships>
            </entity>

            <entity name="DemandSourceAccount" desc="Network account credentials/settings.">
            <fields>id, user_id, demand_source_id, type, is_bidding, extra{}, public_uid</fields>
            <relationships>belongsTo: User, DemandSource; hasMany: LineItem, AppDemandProfile</relationships>
            </entity>

            <entity name="AppDemandProfile" desc="App <-> DemandSourceAccount binding & data.">
            <fields>id, app_id, demand_source_id, account_id, account_type, data{}, public_uid</fields>
            <relationships>belongsTo: App, DemandSource, DemandSourceAccount</relationships>
            </entity>

            <entity name="LineItem" desc="Buy rule / placement per network.">
            <fields>id, human_name, app_id, ad_type, format, bid_floor(decimal-string), account_id, account_type, code, extra(net-specific), public_uid</fields>
            <relationships>belongsTo: App, DemandSourceAccount</relationships>
            </entity>

            <entity name="Segment" desc="Audience filter for targeting.">
            <fields>id, app_id, name, description, enabled, filters[{type,name,operator,values[]}], public_uid</fields>
            <relationships>belongsTo: App; hasMany: AuctionConfigurationV2</relationships>
            </entity>

            <entity name="AuctionConfigurationV2" desc="Auction setup for ad serving (current).">
            <fields>id, name, ad_type, pricefloor(>0), app_id?, ad_unit_ids[]?, segment_id?, bidding[AdapterKey]?, demands[AdapterKey]?, settings{}, timeout_ms?, is_default?, external_win_notifications?, public_uid</fields>
            <relationships>belongsTo: App?, Segment?; references: AdapterKey enums</relationships>
            </entity>

            <entity name="Country" desc="ISO country reference.">
            <fields>id, human_name, alpha_2_code, alpha_3_code</fields>
            <relationships>n/a</relationships>
            </entity>

            <entity name="ResourcesPermissions" desc="RBAC capabilities per resource and instance.">
            <fields>resource: {read, create}; instance: {_permissions.update, _permissions.delete}</fields>
            <relationships>appliesTo: all entities</relationships>
            </entity>
        </entities>

        <types>
            <enum name="AdType" values="banner|interstitial|rewarded"/>
            <enum name="BannerFormat" values="BANNER|LEADERBOARD|MREC|ADAPTIVE"/>
            <enum name="PlatformId" values="ios|android"/>
            <enum name="AdapterKey" values="admob, amazon, applovin, bidmachine, bigoads, chartboost, dtexchange, gam, inmobi, ironsource, meta, mintegral, mobilefuse, moloco, unityads, vkads, vungle, yandex"/>
            <shared name="IDs" values="id(int>0), primary_id(readonly int>0), public_uid(uuid), uuid"/>
            <collectionMeta name="CollectionMeta" values="total_count"/>
        </types>

        <endpointHints>
            <collection preferred="true" for="AppDemandProfile">/api/app_demand_profiles_collection</collection>
            <collection preferred="true" for="LineItem">/api/line_items_collection</collection>
            <collection preferred="true" for="AuctionConfigurationV2">/api/v2/auction_configurations_collection</collection>
            <note>Plural list endpoints (/api/apps, /api/segments, …) return arrays; *collection variants* add filters & meta.</note>
        </endpointHints>
    </adminApiContext>
    """
    spec = fetch_spec()
    spec.setdefault("servers", [{"url": BASE_URL}])
    spec = reduce_spec(filter_get_only(spec))

    conf: Dict[str, Any] = {}
    cookie = None
    if config is not None:
        conf = getattr(config, "configurable", None) or {}
    cookie = (conf.get("X-Admin-Session-Cookie") or conf.get("Cookie"))

    if not cookie:
        return "Missing admin session cookie in config.configurable"

    headers = {"Cookie": cookie, "X-Bidon-App": "web"}
    requests_wrapper = RequestsWrapper(headers=headers)

    llm = ChatAnthropic(
        model="claude-3-7-sonnet-latest",
        api_key=os.getenv("ANTHROPIC_API_KEY"),
        temperature=0,
    )

    api_agent = planner.create_openapi_agent(spec, requests_wrapper, llm, allowed_operations=("GET",), allow_dangerous_requests=True,
                                                 agent_executor_kwargs={
        "max_iterations": 3,              # stop re-planning ad infinitum
        "early_stopping_method": "generate",
        "handle_parsing_errors": True,
        "return_intermediate_steps": True # handy for debugging
        },
        verbose=True,
    )
    # invocation_config: Dict[str, Any] = {"recursion_limit": 2, "configurable": conf}
    result = api_agent.invoke({"input": task})
    return result if isinstance(result, str) else str(result)


@tool
def search_documentation(query: str) -> str:
    """Search documentation for information.

    Args:
        query: The search query string

    Returns:
        A placeholder response (to be implemented later)
    """
    index = LlamaCloudIndex(
        name="bidon-docs",
        project_name="Default",
        organization_id="73814bd1-d8f5-4308-b041-2a55e0b0bfde",
        api_key=os.getenv("LLAMA_INDEX_API_KEY"),
    )
    response = index.as_query_engine().query(query)
    return response.response


def create_agent():
    """Create and configure the React agent with all required components.

    State and memory management is handled automatically by the LangGraph server.
    No custom checkpointer is needed as the platform provides built-in persistence.

    Returns:
        Configured LangGraph agent ready for use
    """
    model = ChatAnthropic(
        model="claude-3-7-sonnet-latest",
        api_key=os.getenv("ANTHROPIC_API_KEY"),
        temperature=0
    )

    tools = [search_documentation, query_admin_api]
    prompt = """
You are a helpful AI assistant for the Bidon Mobile Ads Mediation platform.
Your role is to help users by answering questions about the platform using the available tools to gather information from APIs and documentation.

## Available Tools
You have access to information-gathering tools:

### `query_admin_api` Tool
Use this tool for:
- Querying specific entities and their data
- Fetching monetization setups and configurations  
- Retrieving collections of entities (hint that collection endpoints are preferable when querying multiple items)
- Any other entity-specific data retrieval
# Example: Getting LineItems for app with name "Merge Block iOS"
The admin_api tool accepts natural language task descriptions. You don't need to specify API endpoints - the underlying AI agent will resolve the appropriate endpoints automatically.
If you got an HTTP error - fail fast and return the error message to the user. do not try to retry the request. 

### `search_documentation` Tool  
Use this tool for:
- Finding general information about platform features
- Understanding concepts and terminology
- Getting setup instructions and best practices
- Searching for explanations of how things work

## Information Gathering Approach

Before running tools:
- State in one short, conversational sentence what you plan to do and why
- Gather only the information required to proceed safely
- Stop as soon as you have enough information to provide a well-justified answer

When using tools:
- Use natural language to describe what you need
- Be specific about what entities or information you're looking for
- If you need multiple related pieces of information, explain your approach first

## Output Formatting

Format your response using clear Markdown:
- Use ## ### #### for section headings (never use single #)
- Use bullet points or numbered lists for steps
- Keep paragraphs short and readable
- Bold important terms when first introduced
- Use **bold** or ***bold+italic*** as compact alternatives to headings when appropriate

## Example Usage

If you need to query API data, you might say:
"I'll check the admin API to get the current LineItems for your app to see the monetization setup."

If you need documentation, you might say:  
"Let me search the documentation to understand how ad mediation waterfall works."

Provide your final answer addressing the user's question directly, using the information gathered from the tools.
    """
    agent = create_react_agent(
        model=model,
        tools=tools,
        prompt=prompt,
    )

    return agent

# Create the agent instance
graph = create_agent()
