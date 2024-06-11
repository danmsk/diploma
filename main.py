import grpc
from concurrent import futures
import recomendation_pb2
import recomendation_pb2_grpc
import pandas as pd
import numpy as np
from tensorflow.keras.models import load_model

# Загрузка данных из Excel
ethalon_profiles_df = pd.read_excel('profiles.xlsx', sheet_name='profiles')
directions = ethalon_profiles_df.iloc[:, 0].values
model = load_model('recommendation_model.h5')


class ProfileServiceServicer(recomendation_pb2_grpc.ProfileServiceServicer):
    def GetRecommendations(self, request, context):
        input_profile = np.array(request.profile).reshape(1, -1)

        # Используем модель для предсказаний
        predictions = model.predict(input_profile)

        # Предполагается, что выходные данные модели - это вероятности для каждого направления
        top_indices = np.argsort(predictions[0])[::-1][:8]
        top_directions = directions[top_indices]

        return recomendation_pb2.ProfileResponse(recommendations=list(top_directions))


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    recomendation_pb2_grpc.add_ProfileServiceServicer_to_server(ProfileServiceServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server started at port 50051")
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
