from django.contrib import admin
from django.urls import path, include

from .views import AgentViewSet, RegisterAgent, allocate_address_for_agent
from . import views
urlpatterns = [
    path("admin/", admin.site.urls),

    # Sub-api
    path("provision/", include("api.provision.urls")),
    path('agents/', AgentViewSet.as_view({'get': 'list', 'post': 'create'}), name='agent-list'),
    path('register/', RegisterAgent.as_view(), name='agent-register'),
    path('allocate_address/<int:agent_id>/', views.allocate_address_for_agent, name='allocate_address_for_agent'),
   
    # Autres URL ici

]
