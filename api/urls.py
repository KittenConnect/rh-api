from django.contrib import admin
from django.urls import path

from . import views

urlpatterns = [
    path("", views.index, name="index"),
    path("admin/", admin.site.urls),

    # Sub-api
    path("provision/", include("api.provision.urls"))
]
