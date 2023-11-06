from django.http import HttpResponse, JsonResponse
from django.shortcuts import render
from rest_framework import status, views
from rest_framework.response import Response
from .models import Agent
from .serializers import AgentSerializer
from rest_framework import viewsets
from rest_framework.decorators import action
from .netbox_utils import allocate_loopback, get_prefix
# Create your views here.

def index(request):
    return HttpResponse("Hello world")

# views.py


class RegisterAgent(views.APIView):

    def post(self, request, format=None):
        serializer = AgentSerializer(data=request.data)
        if serializer.is_valid():
            serializer.save()
            return Response(serializer.data, status=status.HTTP_201_CREATED)
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)

class AgentViewSet(viewsets.ModelViewSet):
    queryset = Agent.objects.all()
    serializer_class = AgentSerializer
    
    @action(detail=True, methods=['POST'])
    def connect(self, request, pk=None):
        target_agent_name = request.data.get('target_agent_name')
        

        try:
            target_agent = Agent.objects.get(user=target_agent_name)
        except Agent.DoesNotExist:
            return Response({'error': 'Agent not found'}, status=status.HTTP_404_NOT_FOUND)

        # Logique pour établir la connexion ici, par exemple envoyer les informations à l'agent Y

        return Response({'target_agent': AgentSerializer(target_agent).data})
    
# class to read information from netbox and compile 
# class to send information to netbox and compi
def allocate_address_for_agent(request, agent_id):
    # Remarque: Dans un cas réel, vous obtiendriez l'agent_id d'une manière plus sécurisée, 
    # par exemple à partir d'une authentification
    description = f"Description pour l'agent {agent_id}"

    # Obtenez un /64 à partir du préfixe 10
    prefix_id_64 = 11  # Remplacez par l'ID réel du préfixe dans NetBox
    allocated_subnet_64 = allocate_loopback(prefix_id_64, description)
    
    if not allocated_subnet_64:
        return JsonResponse({'error': 'Impossible d\'allouer le /64'}, status=400)



    # Si tout se passe bien, renvoyez les informations
    return JsonResponse({
        'allocated_subnet_64': allocated_subnet_64,
    })