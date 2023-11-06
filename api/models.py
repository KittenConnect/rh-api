from django.db import models

# Create your models here.

class Agent(models.Model):
    user = models.CharField(max_length=100)  # ou une ForeignKey vers un modèle User
    ip_address = models.GenericIPAddressField()
    public_key = models.CharField(max_length=100)
    class Meta:
        app_label = 'api'  # Remplacez 'api' par le nom de votre application Django